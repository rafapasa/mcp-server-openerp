// internal/dto/whatsapp/webhook.go
package whatsapp

type WebhookRequest struct {
	Object string  `json:"object"`
	Entry  []Entry `json:"entry"`
}

type Entry struct {
	ID      string   `json:"id"`
	Changes []Change `json:"changes"`
}

type Change struct {
	Value Value  `json:"value"`
	Field string `json:"field"`
}

type Value struct {
	MessagingProduct string    `json:"messaging_product"`
	Metadata         Metadata  `json:"metadata"`
	Contacts         []Contact `json:"contacts"`
	Messages         []Message `json:"messages"`
	Statuses         []Status  `json:"statuses"`
}

type Metadata struct {
	PhoneNumberID      string `json:"phone_number_id"`
	DisplayPhoneNumber string `json:"display_phone_number"`
}

type Contact struct {
	Profile Profile `json:"profile"`
	WaID    string  `json:"wa_id"`
}

type Profile struct {
	Name string `json:"name"`
}

type Message struct {
	From        string      `json:"from"`
	ID          string      `json:"id"`
	Timestamp   string      `json:"timestamp"`
	Type        string      `json:"type"`
	Text        MessageText `json:"text"`
	Audio       Media       `json:"audio"`
	Voice       Media       `json:"voice"`
	Image       MediaImage  `json:"image"`
	Interactive Interactive `json:"interactive"`
}

type MessageText struct {
	Body string `json:"body"`
}

type Media struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
}

type MediaImage struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	Caption  string `json:"caption"`
	Sha256   string `json:"sha256"`
}

type Interactive struct {
	Type        string      `json:"type"`
	ButtonReply ButtonReply `json:"button_reply"`
}

type ButtonReply struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Status struct {
	ID           string      `json:"id"`
	Status       string      `json:"status"`
	Timestamp    string      `json:"timestamp"`
	RecipientID  string      `json:"recipient_id"`
	Conversation interface{} `json:"conversation"`
}
