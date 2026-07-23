package commands

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"emote-counter/internal/models"
	"fmt"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const maxBarWidth = 25

func init() {
	handlers.RegisterCommand(models.CommandObject{
		Name:        "leaderboard",
		Description: "Show the emote usage leaderboard",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "channel",
				Description: "Show leaderboard for a specific channel (omit for guild-wide)",
				Type:        discordgo.ApplicationCommandOptionChannel,
				Required:    false,
			},
		},
		Callback: func(props models.CommandProps) {
			sess := props.Sess
			i := props.Interaction

			data := i.ApplicationCommandData()
			channelID := ""
			for _, opt := range data.Options {
				if opt.Name == "channel" {
					channelID = opt.ChannelValue(sess).ID
				}
			}

			var emotes []database.EmoteCount

			if channelID != "" {
				database.DB.
					Where("guild_id = ? AND channel_id = ? AND count > 0", i.GuildID, channelID).
					Order("count desc").
					Find(&emotes)
			} else {
				database.DB.
					Model(&database.EmoteCount{}).
					Select("id, guild_id, name, animated, SUM(count) as count").
					Where("guild_id = ?", i.GuildID).
					Group("id").
					Having("count > 0").
					Order("count desc").
					Find(&emotes)
			}

			if len(emotes) == 0 {
				sess.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: "No emotes have been used yet.",
					},
				})
				return
			}

			maxCount := emotes[0].Count

			var lines []string

			for idx, e := range emotes {
				emoteTag := fmt.Sprintf("<:%s:%s>", e.Name, e.ID)
				if e.Animated {
					emoteTag = fmt.Sprintf("<a:%s:%s>", e.Name, e.ID)
				}

				barWidth := int(math.Round(float64(e.Count) / float64(maxCount) * float64(maxBarWidth)))
				if barWidth == 0 && e.Count > 0 {
					barWidth = 1
				}
				bar := strings.Repeat("█", barWidth)

				line := fmt.Sprintf("**%s** `%s` **%d**", emoteTag, bar, e.Count)
				switch idx {
				case 0:
					line += " 🥇"
				case 1:
					line += " 🥈"
				case 2:
					line += " 🥉"
				}

				lines = append(lines, line)
			}

			leaderboardText := strings.Join(lines, "\n")

			sess.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: leaderboardText,
				},
			})
		},
	})
}
