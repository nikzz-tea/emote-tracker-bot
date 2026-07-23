package events

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm/clause"
)

var emoteRegex = regexp.MustCompile(`<(a?):(\w+):(\d+)>`)

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
			animated := match[1] == "a"
			name := match[2]
			emoteID := match[3]

			database.DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}, {Name: "guild_id"}, {Name: "channel_id"}},
				DoUpdates: clause.Assignments(map[string]any{"count": database.DB.Raw("count + 1")}),
			}).Create(&database.EmoteCount{
				ID:        emoteID,
				GuildID:   m.GuildID,
				ChannelID: m.ChannelID,
				Name:      name,
				Animated:  animated,
				Count:     1,
			})
		}
	})
}
