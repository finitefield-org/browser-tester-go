package dom

import (
	"fmt"
	"strings"
)

var svgAdjustedAttributeNames = map[string]string{
	"attributename":       "attributeName",
	"attributetype":       "attributeType",
	"basefrequency":       "baseFrequency",
	"baseprofile":         "baseProfile",
	"calcmode":            "calcMode",
	"clippathunits":       "clipPathUnits",
	"diffuseconstant":     "diffuseConstant",
	"edgemode":            "edgeMode",
	"filterunits":         "filterUnits",
	"glyphref":            "glyphRef",
	"gradienttransform":   "gradientTransform",
	"gradientunits":       "gradientUnits",
	"kernelmatrix":        "kernelMatrix",
	"kernelunitlength":    "kernelUnitLength",
	"keypoints":           "keyPoints",
	"keysplines":          "keySplines",
	"keytimes":            "keyTimes",
	"lengthadjust":        "lengthAdjust",
	"limitingconeangle":   "limitingConeAngle",
	"markerheight":        "markerHeight",
	"markerunits":         "markerUnits",
	"markerwidth":         "markerWidth",
	"maskcontentunits":    "maskContentUnits",
	"maskunits":           "maskUnits",
	"numoctaves":          "numOctaves",
	"pathlength":          "pathLength",
	"patterncontentunits": "patternContentUnits",
	"patterntransform":    "patternTransform",
	"patternunits":        "patternUnits",
	"pointsatx":           "pointsAtX",
	"pointsaty":           "pointsAtY",
	"pointsatz":           "pointsAtZ",
	"preservealpha":       "preserveAlpha",
	"preserveaspectratio": "preserveAspectRatio",
	"primitiveunits":      "primitiveUnits",
	"refx":                "refX",
	"refy":                "refY",
	"repeatcount":         "repeatCount",
	"repeatdur":           "repeatDur",
	"requiredextensions":  "requiredExtensions",
	"requiredfeatures":    "requiredFeatures",
	"specularconstant":    "specularConstant",
	"specularexponent":    "specularExponent",
	"spreadmethod":        "spreadMethod",
	"startoffset":         "startOffset",
	"stddeviation":        "stdDeviation",
	"stitchtiles":         "stitchTiles",
	"surfacescale":        "surfaceScale",
	"systemlanguage":      "systemLanguage",
	"tablevalues":         "tableValues",
	"targetx":             "targetX",
	"targety":             "targetY",
	"textlength":          "textLength",
	"viewbox":             "viewBox",
	"viewtarget":          "viewTarget",
	"xchannelselector":    "xChannelSelector",
	"ychannelselector":    "yChannelSelector",
	"zoomandpan":          "zoomAndPan",
}

func (s *Store) DumpDOM() string {
	if s == nil {
		return ""
	}
	var b strings.Builder
	for _, childID := range s.documentChildren() {
		s.serializeNode(&b, childID)
	}
	return b.String()
}

func (s *Store) OuterHTMLForNode(id NodeID) (string, error) {
	if s == nil {
		return "", fmt.Errorf("dom store is nil")
	}
	if _, ok := s.nodes[id]; !ok {
		return "", fmt.Errorf("invalid node id: %d", id)
	}
	var b strings.Builder
	s.serializeNode(&b, id)
	return b.String(), nil
}

func (s *Store) TextContentForNode(id NodeID) string {
	if s == nil {
		return ""
	}
	node := s.nodes[id]
	if node == nil {
		return ""
	}
	if node.Kind == NodeKindText {
		return node.Text
	}
	var b strings.Builder
	for _, childID := range node.Children {
		b.WriteString(s.TextContentForNode(childID))
	}
	return b.String()
}

func (s *Store) WholeTextForNode(id NodeID) string {
	if s == nil {
		return ""
	}
	node := s.nodes[id]
	if node == nil || node.Kind != NodeKindText {
		return ""
	}
	if node.Parent == 0 {
		return node.Text
	}
	parent := s.nodes[node.Parent]
	if parent == nil || len(parent.Children) == 0 {
		return node.Text
	}

	index := indexOfNodeID(parent.Children, id)
	if index < 0 {
		return node.Text
	}

	start := index
	for start > 0 {
		prev := s.nodes[parent.Children[start-1]]
		if prev == nil || prev.Kind != NodeKindText {
			break
		}
		start--
	}

	end := index
	for end+1 < len(parent.Children) {
		next := s.nodes[parent.Children[end+1]]
		if next == nil || next.Kind != NodeKindText {
			break
		}
		end++
	}

	var b strings.Builder
	for i := start; i <= end; i++ {
		sibling := s.nodes[parent.Children[i]]
		if sibling == nil || sibling.Kind != NodeKindText {
			continue
		}
		b.WriteString(sibling.Text)
	}
	return b.String()
}

func (s *Store) serializeNode(b *strings.Builder, id NodeID) {
	node := s.nodes[id]
	if node == nil {
		return
	}

	switch node.Kind {
	case NodeKindText:
		if s.shouldSerializeTextRaw(id) {
			b.WriteString(node.Text)
			return
		}
		b.WriteString(escapeTextContent(node.Text))
	case NodeKindElement:
		inSVG := s.isSVGSerializationContext(id)
		b.WriteByte('<')
		b.WriteString(node.TagName)
		for _, attr := range node.Attrs {
			b.WriteByte(' ')
			b.WriteString(serializeAttributeName(attr.Name, inSVG))
			if attr.HasValue {
				b.WriteString(`="`)
				b.WriteString(escapeAttributeValue(attr.Value))
				b.WriteByte('"')
			}
		}
		b.WriteByte('>')

		if isVoidElement(node.TagName) {
			return
		}

		for _, childID := range node.Children {
			s.serializeNode(b, childID)
		}
		b.WriteString("</")
		b.WriteString(node.TagName)
		b.WriteByte('>')
	}
}

func (s *Store) isSVGSerializationContext(id NodeID) bool {
	for current := s.nodes[id]; current != nil; current = s.nodes[current.Parent] {
		if current.Kind == NodeKindElement && current.TagName == "svg" {
			return true
		}
	}
	return false
}

func serializeAttributeName(name string, inSVG bool) string {
	if !inSVG {
		return name
	}
	if adjusted, ok := svgAdjustedAttributeNames[name]; ok {
		return adjusted
	}
	return name
}

func (s *Store) shouldSerializeTextRaw(id NodeID) bool {
	node := s.nodes[id]
	if node == nil || node.Kind != NodeKindText {
		return false
	}
	parent := s.nodes[node.Parent]
	if parent == nil || parent.Kind != NodeKindElement {
		return false
	}
	return parent.TagName == "script"
}

func escapeAttributeValue(value string) string {
	if value == "" {
		return value
	}
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}

func escapeTextContent(value string) string {
	if value == "" {
		return value
	}
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	return value
}
