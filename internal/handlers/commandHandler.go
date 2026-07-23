package handlers

import (
	"emote-counter/internal/models"
	"log"

	"github.com/bwmarrin/discordgo"
)

var commands = make(map[string]models.CommandObject)

func CommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	data := i.ApplicationCommandData()
	cmd, exists := commands[data.Name]
	if !exists {
		return
	}

	user := "unknown"
	if i.Member != nil && i.Member.User != nil {
		user = i.Member.User.Username
	} else if i.User != nil {
		user = i.User.Username
	}

	log.Printf("'%v' used '/%v' command\n", user, data.Name)

	cmd.Callback(models.CommandProps{
		Sess:        s,
		Interaction: i,
	})
}

func RegisterCommand(command models.CommandObject) {
	commands[command.Name] = command
}

func GetCommands() []models.CommandObject {
	var result []models.CommandObject
	for _, cmd := range commands {
		result = append(result, cmd)
	}
	return result
}
