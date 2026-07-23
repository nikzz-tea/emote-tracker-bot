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
		Callback: func(props models.CommandProps) {
			sess := props.Sess
			i := props.Interaction

			var emotes []database.EmoteCount

			database.DB.
				Where("guild_id = ? AND count > 0", i.GuildID).
				Order("count desc").
				Find(&emotes)

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
