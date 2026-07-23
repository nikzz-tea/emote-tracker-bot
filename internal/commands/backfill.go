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
		content := fmt.Sprintf("Error fetching guild emotes: %v", err)
		sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	accumulator := make(map[string]*emoteAccumulator)
	var lastID string
	var totalProcessed int64
	var totalEmotes int64
	startTime := time.Now()

	lastProgressTime := time.Now()

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
			content := fmt.Sprintf("Error fetching messages: %v", err)
			sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			return
		}

		if len(msgs) == 0 {
			break
		}

		for _, msg := range msgs {
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

		if time.Since(lastProgressTime) > 3*time.Second {
			elapsed := time.Since(startTime).Round(time.Second)
			content := fmt.Sprintf("Scanning <#%s>... **%d** messages processed (%d emotes found) — %s elapsed",
				channelID, totalProcessed, totalEmotes, elapsed)
			sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &content,
			})
			lastProgressTime = time.Now()
		}

		time.Sleep(200 * time.Millisecond)
	}

	if len(accumulator) == 0 {
		content := fmt.Sprintf("Scan of <#%s> complete: **0** emotes found in **%d** messages.", channelID, totalProcessed)
		sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		for emoteID, acc := range accumulator {
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}, {Name: "guild_id"}, {Name: "channel_id"}},
				DoUpdates: clause.Assignments(map[string]any{"count": gorm.Expr("count + ?", acc.Count)}),
			}).Create(&database.EmoteCount{
				ID:        emoteID,
				GuildID:   guildID,
				ChannelID: channelID,
				Name:      acc.Name,
				Animated:  acc.Animated,
				Count:     acc.Count,
			}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		content := fmt.Sprintf("Error saving results: %v", err)
		sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &content,
		})
		return
	}

	elapsed := time.Since(startTime).Round(time.Second)
	content := fmt.Sprintf("Scan of <#%s> complete: **%d** emotes found across **%d** messages (%d unique emotes) — %s elapsed",
		channelID, totalEmotes, totalProcessed, len(accumulator), elapsed)
	sess.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: &content,
	})
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
