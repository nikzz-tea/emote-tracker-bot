package handlers

import (
	"emote-counter/internal/models"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const prefix = "!"

var commands = make(map[string]func(models.CommandProps))
var aliases = make(map[string]string)

func CommandHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil {
		return
	}
	if !strings.HasPrefix(m.Content, prefix) {
		return
	}
	if m.Author.ID == s.State.User.ID {
		return
	}

	args := strings.Split(m.Content[len(prefix):], " ")
	commandName := strings.ToLower(args[0])

	if _, exists := aliases[commandName]; exists {
		commandName = aliases[commandName]
	}
	callback, exists := commands[commandName]
	if !exists {
		return
	}

	log.Printf("'%v' used '%v' command\n", m.Author.Username, commandName)

	callback(models.CommandProps{
		Args:    args[1:],
		Sess:    s,
		Message: m,
	})
}

func RegisterCommand(command models.CommandObject) {
	commands[command.Name] = command.Callback

	if command.Aliases != nil {
		for _, alias := range command.Aliases {
			aliases[alias] = command.Name
		}
	}
}
