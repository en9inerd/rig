package feedwatch

import "testing"

func TestExtractParagraphs_Basic(t *testing.T) {
	input := "<p>Hello world</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "Hello world" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_Multiple(t *testing.T) {
	input := "<p>First</p><p>Second</p><p>Third</p>"
	got := extractParagraphs(input)
	if len(got) != 3 {
		t.Fatalf("got %d paragraphs, want 3", len(got))
	}
	want := []string{"First", "Second", "Third"}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestExtractParagraphs_WithAttributes(t *testing.T) {
	input := `<p class="intro">With attrs</p>`
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "With attrs" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_SkipsPre(t *testing.T) {
	input := "<pre>code block</pre><p>Real paragraph</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "Real paragraph" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_SkipsParam(t *testing.T) {
	input := `<param name="x" value="y"><p>After param</p>`
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "After param" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_SkipsPicture(t *testing.T) {
	input := "<picture><source></picture><p>After picture</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "After picture" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_CaseInsensitive(t *testing.T) {
	input := "<P>Upper case</P>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "Upper case" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_MixedCase(t *testing.T) {
	input := "<P class='x'>Mixed</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "Mixed" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_Whitespace(t *testing.T) {
	input := "<p>  \n  trimmed  \n  </p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "trimmed" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_EmptyParagraph(t *testing.T) {
	input := "<p>  </p><p>non-empty</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "non-empty" {
		t.Fatalf("got %v, want [non-empty]", got)
	}
}

func TestExtractParagraphs_AposReplacement(t *testing.T) {
	input := "<p>it&apos;s a test</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "it's a test" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_InnerHTML(t *testing.T) {
	input := `<p>Click <a href="https://example.com">here</a> now</p>`
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != `Click <a href="https://example.com">here</a> now` {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_Multiline(t *testing.T) {
	input := "<p>\nline one\nline two\n</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "line one\nline two" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_NoP(t *testing.T) {
	input := "<div>no paragraphs here</div>"
	got := extractParagraphs(input)
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestExtractParagraphs_EmptyInput(t *testing.T) {
	got := extractParagraphs("")
	if len(got) != 0 {
		t.Fatalf("got %v, want empty", got)
	}
}

func TestExtractParagraphs_UnclosedP(t *testing.T) {
	input := "<p>unclosed paragraph"
	got := extractParagraphs(input)
	if len(got) != 0 {
		t.Fatalf("got %v, want empty (unclosed <p>)", got)
	}
}

func TestExtractParagraphs_NestedContent(t *testing.T) {
	input := "<div><p>inside div</p></div>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "inside div" {
		t.Fatalf("got %v", got)
	}
}

func TestExtractParagraphs_PreThenP(t *testing.T) {
	input := "<pre><code>x := 1</code></pre><p>explanation</p>"
	got := extractParagraphs(input)
	if len(got) != 1 || got[0] != "explanation" {
		t.Fatalf("got %v", got)
	}
}

func TestAtomEntry_URL_Alternate(t *testing.T) {
	e := atomEntry{
		Link: []atomLink{
			{Href: "https://example.com/alt", Rel: "alternate"},
			{Href: "https://example.com/self", Rel: "self"},
		},
	}
	if got := e.URL(); got != "https://example.com/alt" {
		t.Fatalf("URL = %q, want alternate link", got)
	}
}

func TestAtomEntry_URL_EmptyRel(t *testing.T) {
	e := atomEntry{
		Link: []atomLink{
			{Href: "https://example.com/default", Rel: ""},
		},
	}
	if got := e.URL(); got != "https://example.com/default" {
		t.Fatalf("URL = %q, want default link", got)
	}
}

func TestAtomEntry_URL_FallbackFirst(t *testing.T) {
	e := atomEntry{
		Link: []atomLink{
			{Href: "https://example.com/self", Rel: "self"},
			{Href: "https://example.com/enclosure", Rel: "enclosure"},
		},
	}
	if got := e.URL(); got != "https://example.com/self" {
		t.Fatalf("URL = %q, want first link as fallback", got)
	}
}

func TestAtomEntry_URL_NoLinks(t *testing.T) {
	e := atomEntry{}
	if got := e.URL(); got != "" {
		t.Fatalf("URL = %q, want empty", got)
	}
}
