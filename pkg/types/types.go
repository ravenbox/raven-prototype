package types

import "time"

type User struct {
	Id        string
	Name      string
	CreatedAt time.Time
}

type Nest struct {
	Id            string
	Name          string
	StreamChannel StreamChannel
	TextChannel   TextChannel
}

type StreamChannel struct {
	Id   string
	Name string
}

type TextChannel struct {
	Id   string
	Name string
}

type Tree struct {
	Id    string
	Name  string
	Owner User
	Nests []Nest
}
