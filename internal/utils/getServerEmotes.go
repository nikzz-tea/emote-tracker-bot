package utils

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

func GetGuildEmotes(s *discordgo.Session, guildID string) ([]*discordgo.Emoji, error) {
	emotes, err := s.GuildEmojis(guildID)
	if err != nil {
		return []*discordgo.Emoji{}, err
	}

	return emotes, nil
}

func GetGuildEmoteIDs(db *gorm.DB, guildID string) (map[string]bool, error) {
	var ids []string
	err := db.
		Table("emote_counts").
		Where("guild_id = ? AND channel_id = ''", guildID).
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}
