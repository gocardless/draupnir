package routes

type jsonAPIFixture struct {
	Data interface{} `json:"data"`
}

type jsonAPIPayload struct {
	Type       string      `json:"type"`
	ID         string      `json:"id"`
	Attributes interface{} `json:"attributes"`
}

type imageFixture struct {
	BackedUpAt string `json:"backed_up_at"`
	CreatedAt  string `json:"created_at"`
	Ready      bool   `json:"ready"`
	UpdatedAt  string `json:"updated_at"`
}

var listFixture = jsonAPIFixture{
	Data: []jsonAPIPayload{
		{
			Type: "images",
			ID:   "1",
			Attributes: imageFixture{
				BackedUpAt: "2016-01-01T12:33:44Z",
				CreatedAt:  "2016-01-01T12:33:44Z",
				Ready:      false,
				UpdatedAt:  "2016-01-01T12:33:44Z",
			},
		},
	},
}

var createFixture = jsonAPIFixture{
	Data: jsonAPIPayload{
		Type: "images",
		ID:   "1",
		Attributes: imageFixture{
			BackedUpAt: "2016-01-01T12:33:44Z",
			CreatedAt:  "2016-01-01T12:33:44Z",
			Ready:      false,
			UpdatedAt:  "2016-01-01T12:33:44Z",
		},
	},
}
