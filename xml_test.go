package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/kr/pretty"
)

type testCase struct {
	xsd   string
	xml   xmlElem
	gosrc string
}

var (
	tests = []struct {
		exported bool
		prefix   string
		xsd      string
		xml      xmlElem
		gosrc    string
	}{

		{
			false, // Exported structs
			"",    // Struct prefix
			`<schema>
	<element name="titleList" type="titleListType">
	</element>
	<complexType name="titleListType">
		<sequence>
			<element name="title" type="originalTitleType" maxOccurs="unbounded" />
		</sequence>
	</complexType>
	<complexType name="originalTitleType">
		<simpleContent>
			<extension base="titleType">
				<attribute name="original" type="boolean">
				</attribute>
			</extension>
		</simpleContent>
	</complexType>
	<complexType name="titleType">
		<simpleContent>
			<restriction base="textType">
				<maxLength value="300" />
			</restriction>
		</simpleContent>
	</complexType>
	<complexType name="textType">
		<simpleContent>
			<extension base="string">
				<attribute name="language" type="language">
				</attribute>
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xmlElem{
				Name: "titleList",
				Type: "titleList",
				Children: []*xmlElem{
					&xmlElem{
						Name:  "title",
						Type:  "string",
						Cdata: true,
						List:  true,
						Attribs: []xmlAttrib{
							{Name: "language", Type: "string"},
							{Name: "original", Type: "bool"},
						},
					},
				},
			},
			`
type titleList struct {
	Title []title ` + "`xml:\"title\"`" + `
}

type title struct {
	Language string ` + "`xml:\"language,attr\"`" + `
	Original bool ` + "`xml:\"original,attr\"`" + `
	Title    string ` + "`xml:\",chardata\"`" + `
}

				`,
		},

		{
			false, // Exported structs
			"",    // Struct prefix
			`<schema>
	<element name="tagList">
		<complexType>
			<sequence>
				<element name="tag" type="tagReferenceType" minOccurs="0" maxOccurs="unbounded" />
			</sequence>
		</complexType>
	</element>
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="nidType">
				<attribute name="type" type="tagTypeType" use="required" />
			</extension>
		</simpleContent>
	</complexType>
	<simpleType name="nidType">
		<restriction base="string">
			<pattern value="[0-9a-zA-Z\-]+" />
		</restriction>
	</simpleType>
	<simpleType name="tagTypeType">
		<restriction base="string">
		</restriction>
	</simpleType>
</schema>`,
			xmlElem{
				Name: "tagList",
				Type: "tagList",
				Children: []*xmlElem{
					&xmlElem{
						Name:  "tag",
						Type:  "string",
						List:  true,
						Cdata: true,
						Attribs: []xmlAttrib{
							{Name: "type", Type: "string"},
						},
					},
				},
			},
			`
type tagList struct {
	Tag []tag ` + "`xml:\"tag\"`" + `
}

type tag struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	Tag string ` + "`xml:\",chardata\"`" + `
}
			`,
		},

		{
			false, // Exported structs
			"",    // Struct prefix
			`<schema>
				<element name="tagId" type="tagReferenceType" />
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="string">
				<attribute name="type" type="string" use="required" />
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xmlElem{
				Name:  "tagId",
				Type:  "string",
				List:  false,
				Cdata: true,
				Attribs: []xmlAttrib{
					{Name: "type", Type: "string"},
				},
			},
			`
type tagID struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	TagID string ` + "`xml:\",chardata\"`" + `
}
			`,
		},

		{
			true,  // Exported structs
			"xxx", // Struct prefix
			`<schema>
	<element name="tag" type="tagReferenceType" />
	<complexType name="tagReferenceType">
		<simpleContent>
			<extension base="string">
				<attribute name="type" type="string" use="required" />
			</extension>
		</simpleContent>
	</complexType>
</schema>`,
			xmlElem{
				Name:  "tag",
				Type:  "string",
				List:  false,
				Cdata: true,
				Attribs: []xmlAttrib{
					{Name: "type", Type: "string"},
				},
			},
			`
type XxxTag struct {
	Type string ` + "`xml:\"type,attr\"`" + `
	Tagstring ` + "`xml:\",chardata\"`" + `
}
			`,
		},
	}
)

func reset() {
	exported = false
	prefix = ""
	types = make(map[string]struct{})
}

func removeComments(buf bytes.Buffer) bytes.Buffer {
	lines := strings.Split(buf.String(), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "//") {
			lines = append(lines[:i], lines[i+1:]...)
		}
	}
	return *bytes.NewBufferString(strings.Join(lines, "\n"))
}

func TestGenerateGo(t *testing.T) {
	for _, tst := range tests {
		reset()
		exported = tst.exported
		prefix = tst.prefix
		var out bytes.Buffer
		doGenerate(&tst.xml, &out)
		out = removeComments(out)
		if strings.Join(strings.Fields(out.String()), "") != strings.Join(strings.Fields(tst.gosrc), "") {
			t.Errorf("Unexpected generated Go source: %s", tst.xml.Name)
			t.Logf(out.String())
			t.Logf(strings.Join(strings.Fields(out.String()), ""))
			t.Logf(strings.Join(strings.Fields(tst.gosrc), ""))
		}
	}
}

func TestBuildXmlElem(t *testing.T) {
	for _, tst := range tests {
		schema, err := extract(bytes.NewBufferString(tst.xsd))
		if err != nil {
			t.Error(err)
		}
		b := newBuilder([]xsdSchema{schema})
		elems := b.buildXML()
		if len(elems) != 1 {
			t.Errorf("wrong number of xml elements")
		}
		e := elems[0]
		if !reflect.DeepEqual(tst.xml, *e) {
			t.Errorf("Unexpected XML element: %s", e.Name)
			pretty.Println(tst.xml)
			pretty.Println(e)
		}
	}
}
