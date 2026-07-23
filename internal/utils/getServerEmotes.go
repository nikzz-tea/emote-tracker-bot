package utils

import "github.com/bwmarrin/discordgo"

func GetGuildEmotes(s *discordgo.Session, guildID string) ([]*discordgo.Emoji, error) {
	emotes, err := s.GuildEmojis(guildID)
	if err != nil {
		return []*discordgo.Emoji{}, err
	}

	return emotes, nil
}
