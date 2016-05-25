package responses

import (
	"github.com/mitsuse/pushbullet-go/requests"
)

type Push struct {
	Iden                    string  `json:"iden"`
	Created                 float64 `json:"created"`
	Modified                float64 `json:"modified"`
	Active                  bool    `json:"active"`
	Dismissed               bool    `json:"dismissed"`
	SenderIden              string  `json:"sender_iden"`
	SenderEmail             string  `json:"sender_email"`
	SenderEmailNormalized   string  `json:"sender_email_normalized"`
	RecieverIden            string  `json:"reciever_iden"`
	RecieverEmail           string  `json:"reciever_email"`
	RecieverEmailNormalized string  `json:"reciever_email_normalized"`
}

type Note struct {
	*Push
	*requests.Note
}

type Link struct {
	*Push
	*requests.Link
}

type Address struct {
	*Push
	*requests.Address
}

type Checklist struct {
	*Push
	*requests.Checklist
}

type File struct {
	*Push
	*requests.File
}
