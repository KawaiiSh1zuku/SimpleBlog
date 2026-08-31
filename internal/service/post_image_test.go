package service

import "testing"

func TestFirstPostImageURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "markdown inline image",
			content: `intro ![cover](https://cdn.example.com/cover.webp "Cover")`,
			want:    "https://cdn.example.com/cover.webp",
		},
		{
			name:    "markdown angle destination",
			content: `![cover](<https://cdn.example.com/cover%20image.webp>)`,
			want:    "https://cdn.example.com/cover%20image.webp",
		},
		{
			name:    "html image",
			content: `<IMG alt="cover" SRC="https://cdn.example.com/cover.png?x=1&amp;y=2">`,
			want:    "https://cdn.example.com/cover.png?x=1&y=2",
		},
		{
			name: "code and comments are ignored",
			content: "```markdown\n![sample](https://cdn.example.com/code.png)\n```\n" +
				"`<img src=\"https://cdn.example.com/inline.png\">`\n" +
				"<!-- <img src=\"https://cdn.example.com/comment.png\"> -->\n" +
				"![real](https://cdn.example.com/real.png)",
			want: "https://cdn.example.com/real.png",
		},
		{
			name: "first valid absolute image",
			content: `![relative](/relative.png)
<img src="javascript:alert(1)">
![valid](https://cdn.example.com/valid.png)`,
			want: "https://cdn.example.com/valid.png",
		},
		{
			name:    "document order across syntaxes",
			content: `<img src="https://cdn.example.com/first.png"> ![second](https://cdn.example.com/second.png)`,
			want:    "https://cdn.example.com/first.png",
		},
		{
			name: "no supported image",
			content: `![reference][cover]
[cover]: https://cdn.example.com/reference.png`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := firstPostImageURL(test.content)
			if test.want == "" {
				if got != nil {
					t.Fatalf("firstPostImageURL() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != test.want {
				t.Fatalf("firstPostImageURL() = %v, want %q", got, test.want)
			}
		})
	}
}
