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
			{
				Name:        "limit",
				Description: "Number of entries to show (0 = all, default: 10)",
				Type:        discordgo.ApplicationCommandOptionInteger,
				Required:    false,
			},
		},
		Callback: func(props models.CommandProps) {
			sess := props.Sess
			i := props.Interaction

			data := i.ApplicationCommandData()
			channelID := ""
			limit := 10
			for _, opt := range data.Options {
				switch opt.Name {
				case "channel":
					channelID = opt.ChannelValue(sess).ID
				case "limit":
					limit = int(opt.IntValue())
				}
			}

			var emotes []database.EmoteCount

			if channelID != "" {
				q := database.DB.
					Where("guild_id = ? AND channel_id = ? AND count > 0", i.GuildID, channelID).
					Order("count desc")
				if limit > 0 {
					q = q.Limit(limit)
				}
				q.Find(&emotes)
			} else {
				if limit > 0 {
					database.DB.Raw(
						"SELECT id, guild_id, name, animated, SUM(count) AS count FROM emote_counts WHERE guild_id = ? AND channel_id != '' GROUP BY id HAVING SUM(count) > 0 ORDER BY count DESC LIMIT ?",
						i.GuildID, limit,
					).Scan(&emotes)
				} else {
					database.DB.Raw(
						"SELECT id, guild_id, name, animated, SUM(count) AS count FROM emote_counts WHERE guild_id = ? AND channel_id != '' GROUP BY id HAVING SUM(count) > 0 ORDER BY count DESC",
						i.GuildID,
					).Scan(&emotes)
				}
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
