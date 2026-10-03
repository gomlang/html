package native

import (
    native_0 "example.com/goml-ecosystem/html/adapter"
)

func GomlBind_attribute_name(p0 native_0.Attribute) string {
    return native_0.AttributeName(p0)
}

func GomlBind_attribute_namespace(p0 native_0.Attribute) string {
    return native_0.AttributeNamespace(p0)
}

func GomlBind_attribute_value(p0 native_0.Attribute) string {
    return native_0.AttributeValue(p0)
}

func GomlBind_attribute_value_by_name(p0 native_0.Node, p1 string, p2 string) (string, bool) {
    return native_0.AttributeValueByName(p0, p1, p2)
}

func GomlBind_attributes(p0 native_0.Node) []native_0.Attribute {
    return native_0.Attributes(p0)
}

func GomlBind_data(p0 native_0.Node) string {
    return native_0.Data(p0)
}

func GomlBind_error_code(p0 error) int {
    return native_0.ErrorCode(p0)
}

func GomlBind_error_message(p0 error) string {
    return native_0.ErrorMessage(p0)
}

func GomlBind_first_child(p0 native_0.Node) native_0.Node {
    return native_0.FirstChild(p0)
}

func GomlBind_is_nil(p0 native_0.Node) bool {
    return native_0.IsNil(p0)
}

func GomlBind_kind(p0 native_0.Node) int {
    return native_0.Kind(p0)
}

func GomlBind_matches(p0 native_0.Node, p1 string, p2 int) (bool, error) {
    return native_0.Matches(p0, p1, p2)
}

func GomlBind_namespace(p0 native_0.Node) string {
    return native_0.Namespace(p0)
}

func GomlBind_next_sibling(p0 native_0.Node) native_0.Node {
    return native_0.NextSibling(p0)
}

func GomlBind_parent(p0 native_0.Node) native_0.Node {
    return native_0.Parent(p0)
}

func GomlBind_parse(p0 string, p1 int, p2 int, p3 int) (native_0.Node, error) {
    return native_0.Parse(p0, p1, p2, p3)
}

func GomlBind_parse_fragment(p0 string, p1 string, p2 int, p3 int, p4 int) (native_0.Node, error) {
    return native_0.ParseFragment(p0, p1, p2, p3, p4)
}

func GomlBind_render(p0 native_0.Node, p1 bool, p2 int) (string, error) {
    return native_0.Render(p0, p1, p2)
}

func GomlBind_same_node(p0 native_0.Node, p1 native_0.Node) bool {
    return native_0.SameNode(p0, p1)
}

func GomlBind_sanitize(p0 string, p1 bool, p2 int, p3 int, p4 int, p5 int) (string, error) {
    return native_0.Sanitize(p0, p1, p2, p3, p4, p5)
}

func GomlBind_select(p0 native_0.Node, p1 string, p2 int, p3 int, p4 bool) ([]native_0.Node, error) {
    return native_0.Select(p0, p1, p2, p3, p4)
}

func GomlBind_text(p0 native_0.Node, p1 int) (string, error) {
    return native_0.Text(p0, p1)
}
