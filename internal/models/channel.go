package models

type Channel struct {
	ID                 int    `db:"id" json:"id"`
	Name               string `db:"name" json:"name"`
	Description        string `db:"description" json:"description"`
	ChannelPicturePath string `db:"channel_picture_path" json:"channel_picture_path"`
}
