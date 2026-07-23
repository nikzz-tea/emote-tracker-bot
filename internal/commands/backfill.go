package commands

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"emote-counter/internal/models"
	"emote-counter/internal/utils"
	"fmt"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	backfillMu       sync.Mutex
	backfillActiveCh string
)

type emoteAccumulator struct {
	Name     string
	Animated bool
	Count    int64
}

func init() {
	handlers.RegisterCommand(models.CommandObject{
		Name:        "backfill",
		Description: "Scan channel history to count past emote usage",
		AdminOnly:   true,
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "channel",
				Description: "The channel to scan",
				Type:        discordgo.ApplicationCommandOptionChannel,
				Required:    true,
			},
			{
				Name:        "limit",
				Description: "Max messages to scan (default: all)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    false,
			},
		},
		Callback: func(props models.CommandProps) {
			sess := props.Sess
			i := props.Interaction

			data := i.ApplicationCommandData()

			var channelID string
			var limit int64
			for _, opt := range data.Options {
				switch opt.Name {
				case "channel":
					channelID = opt.ChannelValue(sess).ID
				case "limit":
					limit = opt.IntValue()
				}
			}

			ch, err := sess.State.Channel(channelID)
			if err != nil {
				ch, err = sess.Channel(channelID)
			}
			if err != nil || ch.Type != discordgo.ChannelTypeGuildText {
				respondError(sess, i, "Channel must be a text channel in this server.")
				return
			}

			backfillMu.Lock()
			if backfillActiveCh != "" {
				backfillMu.Unlock()
				respondError(sess, i, fmt.Sprintf("A backfill is already running on <#%s>.", backfillActiveCh))
				return
			}
			backfillActiveCh = channelID
			backfillMu.Unlock()

			sess.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
			})

			go runBackfill(sess, i, channelID, limit)
		},
	})
}

func runBackfill(sess *discordgo.Session, i *discordgo.InteractionCreate, channelID string, limit int64) {
	defer func() {
		backfillMu.Lock()
		backfillActiveCh = ""
		backfillMu.Unlock()
	}()

	guildID := i.GuildID
	valid, err := utils.GetGuildEmoteIDs(database.DB, guildID)
	if err != nil {
		editResponse(sess, i, fmt.Sprintf("Error fetching guild emotes: %v", err))
		return
	}

	var cp database.BackfillCheckpoint
	resuming := database.DB.Where("channel_id = ? AND guild_id = ?", channelID, guildID).First(&cp).Error == nil

	accumulator := make(map[string]*emoteAccumulator)
	var lastID string
	var totalProcessed int64
	var totalEmotes int64
	startTime := time.Now()

	if resuming {
		lastID = cp.LastMessageID
		totalProcessed = cp.TotalProcessed
		totalEmotes = cp.TotalEmotes
	}

	lastFlushTime := time.Now()
	lastProgressTime := time.Now()

	flush := func() {
		if len(accumulator) == 0 {
			return
		}
		err := database.DB.Transaction(func(tx *gorm.DB) error {
			for emoteID, acc := range accumulator {
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "id"}, {Name: "guild_id"}, {Name: "channel_id"}},
					DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + ?", acc.Count)}),
				}).Create(&database.EmoteCount{
					ID:        emoteID,
					GuildID:   guildID,
					ChannelID: channelID,
					Name:      acc.Name,
					Animated:  acc.Animated,
					Count:     acc.Count,
				}).Error; err != nil {
					return err
				}
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "channel_id"}, {Name: "guild_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"last_message_id", "total_processed", "total_emotes"}),
			}).Create(&database.BackfillCheckpoint{
				ChannelID:      channelID,
				GuildID:        guildID,
				LastMessageID:  lastID,
				TotalProcessed: totalProcessed,
				TotalEmotes:    totalEmotes,
			}).Error
		})
		if err != nil {
			editResponse(sess, i, fmt.Sprintf("Error saving progress: %v", err))
		}
		accumulator = make(map[string]*emoteAccumulator)
	}

	for {
		batchSize := 100
		if limit > 0 {
			remaining := limit - totalProcessed
			if remaining <= 0 {
				break
			}
			if remaining < 100 {
				batchSize = int(remaining)
			}
		}

		msgs, err := sess.ChannelMessages(channelID, batchSize, lastID, "", "")
		if err != nil {
			flush()
			editResponse(sess, i, fmt.Sprintf("Error fetching messages (progress saved): %v", err))
			return
		}

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
			if msg.Author != nil && msg.Author.Bot {
				continue
			}
			emotes := utils.ExtractEmotes(msg.Content)
			for _, e := range emotes {
				if !valid[e.ID] {
					continue
				}
				acc, ok := accumulator[e.ID]
				if !ok {
					acc = &emoteAccumulator{
						Name:     e.Name,
						Animated: e.Animated,
					}
					accumulator[e.ID] = acc
				}
				acc.Count++
				totalEmotes++
			}
		}

		totalProcessed += int64(len(msgs))
		lastID = msgs[len(msgs)-1].ID

		if time.Since(lastFlushTime) > 30*time.Second {
			flush()
			lastFlushTime = time.Now()
		}

		if time.Since(lastProgressTime) > 3*time.Second {
			elapsed := time.Since(startTime).Round(time.Second)
			content := fmt.Sprintf("Scanning <#%s>... **%d** messages processed (%d emotes found) — %s elapsed",
				channelID, totalProcessed, totalEmotes, elapsed)
			editResponse(sess, i, content)
			lastProgressTime = time.Now()
		}

		time.Sleep(200 * time.Millisecond)
	}

	flush()
	database.DB.Where("channel_id = ? AND guild_id = ?", channelID, guildID).Delete(&database.BackfillCheckpoint{})

	elapsed := time.Since(startTime).Round(time.Second)
	content := fmt.Sprintf("Scan of <#%s> complete: **%d** emotes found across **%d** messages — %s elapsed",
		channelID, totalEmotes, totalProcessed, elapsed)
	editResponse(sess, i, content)
}

func respondError(sess *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	sess.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func editResponse(sess *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
}
