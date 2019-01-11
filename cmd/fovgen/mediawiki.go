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

The wide tables keep track of aspect ratios, such that any format you like can fit into any target format. In other words the equivalent format is strictly wider, so from the target (new) format you might need to crop to get the original framing. This is the reason why these numbers are different than other numbers usually circulated in photographic communities. For example, a 300mm lens on 8x10 is a normal lens, and "feels" (more or less) like a 50mm or a 45mm lens on 35mm (full frame). However, you need a 35mm lens on full frame that you will then crop to a 5:4 ratio to get the same framing as the 300mm lens on large format.

The horizontal tables only take into account the horizontal field of view.

The vertical tables only take into account the vertical field of view.

Many equivalency tables take into account the diagonal field of view, but I think that's a pointless comparison. You usually have some lens/camera combination and ask yourself ''"what lens do I need on this other format so that after cropping I get the same framing?"''. These tables answer this question.

The 6x10 format is actually 8x10 used with lenses that don't quite have the coverage for 8x10 (ultra-ultrawides). It's widely used by Clyde Butcher, so it's included here. For 6x10 and 12x20, only lenses known to be used by Clyde Butcher are included.

The tilt/shift columns indicate what focal length you need such that when stitching you could get the original framing. We assume the Canon lenses have a 67mm image circle (from the Canon 17mm and 24mm press release). By pure geometry, for 12mm of shift the image circle has to be at least 64.6mm, and for 11mm of shift it has to be 62.7mm.

== How to use these tables ==

First decide what format and what crop you like to shoot already. Go to the corresponding section, then look for the column with the format you are interested in trying next and find the focal length.

For example, say you like 50mm on full frame, and you like shooting in 4:3 aspect ratio. You are interested in trying out 4x5 large format. First go to "35mm full frame (4:3 crop)" section, then find the row with 50mm and look for the column with "4x5". You will find 198mm, so you need a 200mm lens. 4x5 does not have a 4:3 aspect ratio, so with this 200mm you will have to crop from the top to get the original framing you like.

Another example, say you like 300mm on 8x10 and would like to shoot Fuji GFX medium format. First go to "Large format (8x10)" section and find the row with 300mm, then look for the column with "Fuji GFX". You will find that you need a 48mm lens. Again the aspect ration is different. Fuji is slightly longer, so you will need to crop from the sides to get the original framing, but 48mm will be good.

Alternatively, say that instead of Fuji GFX you want to shoot Hasselblad H5D-60 with the tilt-shft adapter. In the same 300mm row you found earlier, look for "HxD-60/100 (HTS)". You will find that you need a 40mm Hasselblad lens.

=== Stitched images with Tilt/Shift lenses ===

Say you like shooting 90mm on 6x10, and want to get the same framing via stitching on full frame or medium format (doesn't matter) with full frame tilt/shift lenses. You go to the "Large format (6x10)" section, find the row with 90mm, and look for "Canon TS (12mm)". You will find out you that need a 19mm TS lens. You either hope the Nikon 19mm TS lens has a 67mm image circle, or you go with the 17mm Canon.

== Focal lens equivalents ==

{{$sensors := .Sensors}}
{{$tslenses := .TiltShifts}}

{{range .CameraInfo}}
=== {{.Name}} ===

{| class="wikitable"
|+ {{.Name}} (wide)
|-
|Focal legth
|Horizontal FOV
|Vertical FOV
{{- range $sensors}}
|{{.}}
{{- end}}
{{- range $tslenses}}
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
{{- range $ts := $tslenses}}
|{{index $li.EqTS $ts | printf "%.0f"}}
{{- end}}
{{- end}}
|}

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
