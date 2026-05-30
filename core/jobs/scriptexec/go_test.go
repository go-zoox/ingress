package scriptexec

import (
	"strings"
	"testing"
)

func TestSplitGoImports(t *testing.T) {
	imports, body := splitGoImports(`import "fmt"

fmt.Println("hi")`)
	if imports != `import "fmt"` {
		t.Fatalf("imports = %q", imports)
	}
	if body != `fmt.Println("hi")` {
		t.Fatalf("body = %q", body)
	}
}

func TestSplitGoImports_NoImport(t *testing.T) {
	imports, body := splitGoImports(`fmt.Println("hi")`)
	if imports != "" {
		t.Fatalf("imports = %q", imports)
	}
	if body != `fmt.Println("hi")` {
		t.Fatalf("body = %q", body)
	}
}

func TestWrapGoScript(t *testing.T) {
	got := wrapGoScript(`import "fmt"

fmt.Println("hi")`)
	want := `import "fmt"
func __run() {
fmt.Println("hi")
}`
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("wrap = %q", got)
	}

	got = wrapGoScript(`fmt.Println("plain")`)
	want = `func __run() {
fmt.Println("plain")
}`
	if strings.TrimSpace(got) != strings.TrimSpace(want) {
		t.Fatalf("wrap no import = %q", got)
	}
}

func TestSplitGoImports_ParenBlock(t *testing.T) {
	imports, body := splitGoImports(`import (
	"fmt"
	"strings"
)

fmt.Println(strings.ToUpper("hi"))`)
	if !strings.Contains(imports, `"fmt"`) || !strings.Contains(imports, `"strings"`) {
		t.Fatalf("imports = %q", imports)
	}
	if !strings.Contains(body, `strings.ToUpper`) {
		t.Fatalf("body = %q", body)
	}
}
