package models

import "github.com/bwmarrin/discordgo"

type CommandObject struct {
	Name        string
	Description string
	Options     []*discordgo.ApplicationCommandOption
	Callback    func(CommandProps)
}

type CommandProps struct {
	Sess        *discordgo.Session
	Interaction *discordgo.InteractionCreate
}
