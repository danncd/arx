package tools

import "encoding/json"

func generationDefinitions() []Definition {
	return []Definition{
		{"generate", "Create an image, video, or speech using the user's installed default model. Choose the operation from their request. Do not download models or claim success before the result. Source is an arx-media reference returned by tools or an arx-image reference attached by the user for image editing or image-to-video. Image generation accepts one reference image through source, or an ordered list through sources (not both). FLUX.2 Klein accepts up to four sources; Z-Image Turbo, SD Turbo, DreamShaper, and SD 1.5 accept only one. Pass multiple references as a JSON array, for example: sources: [\"arx-media:<first-id>\", \"arx-media:<second-id>\"]. Never join references into a comma-separated source string. Explain the role of each reference in the prompt. Use an attached image or a previously generated image as the reference when it helps fulfill the request; describe which visual details to preserve and which to change. Use files image to inspect generated images when useful. Omit voice to use the selected speech model default. If a requested voice is unsupported, the error lists available voices. Some video models support text only; do not assume image-to-video support. Default video is a short silent clip; use speech and media to add narration. Lip sync is not supported. Speech prompt is the exact text to speak. Generated images are not shown automatically. Display each generated image in your reply using the exact ![description](arx-media:<id>) Markdown from the result. Do not change the reference prefix or ID. Display requires no inspection step. Video and speech have automatic playback controls. Pixels are not automatically sent to your vision input.", json.RawMessage(`{
  "type": "object",
  "properties": {
    "operation": {
      "type": "string",
      "enum": [
        "image",
        "video",
        "speech"
      ]
    },
    "prompt": {
      "type": "string"
    },
    "source": {
      "type": "string",
      "description": "Exactly one saved image reference. Never comma-separate IDs. For multiple images use sources instead."
    },
    "sources": {
      "type": "array",
      "description": "Ordered image reference IDs, for example [\"arx-media:<id1>\",\"arx-media:<id2>\"]. Omit source. FLUX.2 Klein allows up to 4; other current image models allow 1.",
      "items": {
        "type": "string"
      },
      "minItems": 1,
      "maxItems": 4
    },
    "width": {
      "type": "integer"
    },
    "height": {
      "type": "integer"
    },
    "seed": {
      "type": "integer"
    },
    "voice": {
      "type": "string"
    },
    "speed": {
      "type": "number"
    },
    "frames": {
      "type": "integer"
    }
  },
  "required": [
    "operation",
    "prompt"
  ],
  "additionalProperties": false
}`)},
		{"media", "Combine a saved video and speech output. Source and audio must be media references returned by tools. This adds narration without synchronizing mouth movement. The shorter input determines output duration.", json.RawMessage(`{
  "type": "object",
  "properties": {
    "operation": {
      "type": "string",
      "enum": [
        "combine"
      ]
    },
    "source": {
      "type": "string"
    },
    "audio": {
      "type": "string"
    }
  },
  "required": [
    "operation",
    "source",
    "audio"
  ],
  "additionalProperties": false
}`)},
	}
}
