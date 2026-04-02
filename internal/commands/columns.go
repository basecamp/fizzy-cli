package commands

import "github.com/basecamp/fizzy-cli/internal/render"

// Column definitions for styled/markdown table rendering of each entity type.
var (
	boardColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
	}

	cardColumns = render.Columns{
		{Header: "#", Field: "number"},
		{Header: "Title", Field: "title"},
	}

	columnColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
	}

	stepColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Content", Field: "content"},
		{Header: "Done", Field: "completed"},
	}

	commentColumns = render.Columns{
		{Header: "ID", Field: "id"},
	}

	tagColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Title", Field: "title"},
	}

	userColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
	}

	authProfileColumns = render.Columns{
		{Header: "Profile", Field: "profile"},
		{Header: "Active", Field: "active"},
		{Header: "Board", Field: "board"},
		{Header: "Base URL", Field: "base_url"},
	}

	notificationColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Message", Field: "message"},
		{Header: "Read", Field: "read"},
	}

	pinColumns = render.Columns{
		{Header: "#", Field: "number"},
		{Header: "Title", Field: "title"},
	}

	reactionColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Content", Field: "content"},
	}

	searchColumns = cardColumns

	attachmentColumns = render.Columns{
		{Header: "#", Field: "index"},
		{Header: "Filename", Field: "filename"},
		{Header: "Type", Field: "content_type"},
		{Header: "Size", Field: "filesize"},
	}

	webhookColumns = render.Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
		{Header: "URL", Field: "payload_url"},
		{Header: "Active", Field: "active"},
	}
)
