package database

type EmoteCount struct {
	ID      string `gorm:"primaryKey"`
	GuildID string `gorm:"primaryKey"`
	Name    string
	Count   int64 `gorm:"default:0"`
}
