package events

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"regexp"

	"github.com/bwmarrin/discordgo"
)

var emoteRegex = regexp.MustCompile(`<a?:\w+:(\d+)>`)

func init() {
	handlers.RegisterEvent(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.ID == s.State.User.ID {
			return
		}

		matches := emoteRegex.FindAllStringSubmatch(m.Content, -1)
		if len(matches) == 0 {
			return
		}

		for _, match := range matches {
			emoteID := match[1]

			database.DB.
				Model(&database.EmoteCount{}).
				Where("id = ? AND guild_id = ?", emoteID, m.GuildID).
				Update("count", database.DB.Raw("count + 1"))
		}
	})
}
