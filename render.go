package main

import "image/color"

const (
	ViewW    = 480
	ViewH    = 360
	VelScale = 1.0
)

// CameraEye is shared with back-face culling in the rasterizer.
var CameraEye = Vec3{0, 1.35, 2.9}

var (
	viewMat = LookAt(CameraEye, Vec3{0, 1.2, 0}, Vec3{0, 1, 0})
	projMat = Perspective(1.15, float64(ViewW)/float64(ViewH), 0.1, 20)
	vpMat   = projMat.Mul(viewMat)
)

// project returns screen x/y, NDC z and 1/w for a skinned world vertex.
func project(p Vec3) (sx, sy, ndcz, invw float64) {
	c := vpMat.M
	w := c[12]*p.X + c[13]*p.Y + c[14]*p.Z + c[15]
	invw = 1 / w
	cx := (c[0]*p.X + c[1]*p.Y + c[2]*p.Z + c[3]) * invw
	cy := (c[4]*p.X + c[5]*p.Y + c[6]*p.Z + c[7]) * invw
	cz := (c[8]*p.X + c[9]*p.Y + c[10]*p.Z + c[11]) * invw
	sx = (cx + 1) * 0.5 * ViewW
	sy = (1 - cy) * 0.5 * ViewH
	return sx, sy, cz, invw
}

// RenderFrame skins every vertex twice (current and previous bone poses)
// so each triangle carries true per-vertex post-skinning screen velocity.
func RenderFrame(mesh *Mesh, sk *Skeleton, pose, prev PoseParams) *Frame {
	world := sk.WorldPose(pose)
	prevWorld := sk.WorldPose(prev)
	f := NewFrame(ViewW, ViewH)
	f.Clear(color.RGBA{14, 16, 24, 255})

	vout := make([]VOut, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		cur := SkinVertex(v.Pos, v.Weights, world, sk)
		old := SkinVertex(v.Pos, v.Weights, prevWorld, sk)
		sx, sy, cz, iw := project(cur)
		ox, oy, _, _ := project(old)
		vout[i] = VOut{
			SX: sx, SY: sy, NDCZ: cz, InvW: iw,
			Color: v.Color, World: cur,
			VelX: (sx - ox) * VelScale,
			VelY: (sy - oy) * VelScale,
		}
	}
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		f.DrawTriangle(vout[mesh.Indices[t]], vout[mesh.Indices[t+1]], vout[mesh.Indices[t+2]])
	}
	return f
}

// RenderRootBaseline skins geometry correctly but assigns the root
// bone's screen-space velocity to every vertex. It reproduces the
// classic bug where a waving arm inherits only pelvis motion and
// therefore leaves a ghost trail in the history buffer.
func RenderRootBaseline(mesh *Mesh, sk *Skeleton, pose, prev PoseParams) *Frame {
	world := sk.WorldPose(pose)
	prevWorld := sk.WorldPose(prev)
	root := world[0].TransformPoint(Vec3{})
	oldRoot := prevWorld[0].TransformPoint(Vec3{})
	rx, ry, _, _ := project(root)
	ox, oy, _, _ := project(oldRoot)
	rootVX, rootVY := rx-ox, ry-oy

	f := NewFrame(ViewW, ViewH)
	f.Clear(color.RGBA{14, 16, 24, 255})
	vout := make([]VOut, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		cur := SkinVertex(v.Pos, v.Weights, world, sk)
		sx, sy, cz, iw := project(cur)
		vout[i] = VOut{
			SX: sx, SY: sy, NDCZ: cz, InvW: iw,
			Color: v.Color, World: cur,
			VelX: rootVX,
			VelY: rootVY,
		}
	}
	for t := 0; t+2 < len(mesh.Indices); t += 3 {
		f.DrawTriangle(vout[mesh.Indices[t]], vout[mesh.Indices[t+1]], vout[mesh.Indices[t+2]])
	}
	return f
}

// DrawHud paints static overlay UI after the scene. It never touches
// Depth/Cover/VelX/VelY, which keeps HUD pixels out of the velocity map.
func DrawHud(f *Frame) {
	white := color.RGBA{235, 238, 245, 255}
	green := color.RGBA{90, 220, 150, 255}
	dark := color.RGBA{20, 26, 38, 235}
	accent := color.RGBA{240, 190, 70, 255}

	f.DrawHudRect(14, 14, 158, 86, dark)
	f.DrawHudRect(14, 14, 158, 19, accent)
	f.DrawHudText("HUD STATIC", 20, 21, 1, color.RGBA{20, 20, 20, 255})
	f.DrawHudText("VEL BUFFER", 22, 34, 1, white)
	f.DrawHudText("NOT WRITTEN", 22, 46, 1, white)
	f.DrawHudRect(22, 64, 140, 72, color.RGBA{60, 70, 90, 255})
	f.DrawHudRect(22, 64, 22+96, 72, green)

	f.DrawHudRect(ViewW-132, ViewH-40, ViewW-16, ViewH-16, dark)
	f.DrawHudText("UI OVERLAY", ViewW-124, ViewH-33, 1, white)
}
