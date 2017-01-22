package routes

import "github.com/google/jsonapi"

var listImagesFixture = jsonapi.ManyPayload{
	Data: []*jsonapi.Node{
		{
			Type: "images",
			ID:   "1",
			Attributes: map[string]interface{}{
				"backed_up_at": "2016-01-01T12:33:44Z",
				"created_at":   "2016-01-01T12:33:44Z",
				"ready":        false,
				"updated_at":   "2016-01-01T12:33:44Z",
			},
		},
	},
}

var createImageFixture = jsonapi.OnePayload{
	Data: &jsonapi.Node{
		Type: "images",
		ID:   "1",
		Attributes: map[string]interface{}{
			"backed_up_at": "2016-01-01T12:33:44Z",
			"created_at":   "2016-01-01T12:33:44Z",
			"ready":        false,
			"updated_at":   "2016-01-01T12:33:44Z",
		},
	},
}

var doneImageFixture = jsonapi.OnePayload{
	Data: &jsonapi.Node{
		Type: "images",
		ID:   "1",
		Attributes: map[string]interface{}{
			"backed_up_at": "2016-01-01T12:33:44Z",
			"created_at":   "2016-01-01T12:33:44Z",
			"ready":        true,
			"updated_at":   "2016-01-01T12:33:44Z",
		},
	},
}

var getImageFixture = jsonapi.OnePayload{
	Data: &jsonapi.Node{
		Type: "images",
		ID:   "1",
		Attributes: map[string]interface{}{
			"backed_up_at": "2016-01-01T12:33:44Z",
			"created_at":   "2016-01-01T12:33:44Z",
			"ready":        false,
			"updated_at":   "2016-01-01T12:33:44Z",
		},
	},
}

var createInstanceFixture = jsonapi.OnePayload{
	Data: &jsonapi.Node{
		Type: "instances",
		ID:   "1",
		Attributes: map[string]interface{}{
			"image_id":   1,
			"created_at": "2016-01-01T12:33:44Z",
			"updated_at": "2016-01-01T12:33:44Z",
		},
	},
}
