package models

type Tag struct {
	Id  int    `db:"id" json:"id"`
	Tag string `db:"tag" json:"tag"`
}
