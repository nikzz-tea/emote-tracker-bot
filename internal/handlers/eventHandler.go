package handlers

import (
	"github.com/bwmarrin/discordgo"
)

var events = []any{}

func EventHandler(s *discordgo.Session) {

	for _, event := range events {
		s.AddHandler(event)
	}
}

func RegisterEvent(event any) {
	events = append(events, event)
}
