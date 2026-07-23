package events

import (
	"emote-counter/internal/database"
	"emote-counter/internal/handlers"
	"emote-counter/internal/utils"

	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm/clause"
)

func init() {
	handlers.RegisterEvent(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author == nil || m.Author.Bot {
			return
		}

		emotes := utils.ExtractEmotes(m.Content)
		if len(emotes) == 0 {
			return
		}

		valid, _ := utils.GetGuildEmoteIDs(database.DB, m.GuildID)

		for _, e := range emotes {
			if !valid[e.ID] {
				continue
			}
			database.DB.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}, {Name: "guild_id"}, {Name: "channel_id"}},
				DoUpdates: clause.Assignments(map[string]any{"count": database.DB.Raw("count + 1")}),
			}).Create(&database.EmoteCount{
				ID:        e.ID,
				GuildID:   m.GuildID,
				ChannelID: m.ChannelID,
				Name:      e.Name,
				Animated:  e.Animated,
				Count:     1,
			})
		}
	})
}
