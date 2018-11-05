// Copyright (c) 2018 Aram Hăvărneanu <aram@mgk.ro>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package main

import (
	"text/template"
)

var wiki = template.Must(template.New("wiki").Parse(wikiPage))

const wikiPage = `
== Introduction ==

The tables below lists the focal length equivalents between different formats.

The wide tables are keeping track of aspect ratios, such that every format can fit into any target format. In other words the equivalent format is strictly wider, so from the target format you might need to crop to get the original framing.

The horizontal tables only take into account the horizontal field of view.

The vertical tables only take into account the vertical field of view.

== Focal lens equivalents ==

{{$sensors := .Sensors}}

{{range .CameraInfo}}
=== {{.Name}} ===

==== Wide ====
{| class="wikitable"
|+ {{.Name}} (wide)
|-
|Focal legth
|Horizontal FOV
|Vertical FOV
{{- range $sensors}}
|{{.}}
{{- end}}
{{- range .Lenses }}
|-
|{{.Lens}}
|{{printf "%.1f°" .HFoV}}
|{{printf "%.1f°" .VFoV}} {{$li := .}}
{{- range $s := $sensors}}
|{{index $li.EqW $s | printf "%.0f"}}
{{- end}}
{{- end}}
|}

==== Horizontal ====
{| class="wikitable"
|+ {{.Name}} (horizontal)
|-
|Focal legth
|Horizontal FOV
|Vertical FOV
{{- range $sensors}}
|{{.}}
{{- end}}
{{- range .Lenses }}
|-
|{{.Lens}}
|{{printf "%.1f°" .HFoV}}
|{{printf "%.1f°" .VFoV}} {{$li := .}}
{{- range $s := $sensors}}
|{{index $li.EqH $s | printf "%.0f"}}
{{- end}}
{{- end}}
|}

==== Vertical ====
{| class="wikitable"
|+ {{.Name}} (vertical)
|-
|Focal legth
|Horizontal FOV
|Vertical FOV
{{- range $sensors}}
|{{.}}
{{- end}}
{{- range .Lenses }}
|-
|{{.Lens}}
|{{printf "%.1f°" .HFoV}}
|{{printf "%.1f°" .VFoV}} {{$li := .}}
{{- range $s := $sensors}}
|{{index $li.EqV $s | printf "%.0f"}}
{{- end}}
{{- end}}
|}

{{end}}

== Code ==

This page was generated by https://mgk.ro/cmd/fovgen.
`