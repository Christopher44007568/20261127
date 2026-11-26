package main

import "fmt"

// RenderCounterZoom compares regular / adopted-bold / rejected-overbold.
func RenderCounterZoom(fs *FontSet) *Scene {
	const W, H = 1720, 1240
	sc := NewScene(W, H, 3, CPaper)
	sc.Title("局部放大：粗字重不得吞掉字腔（counter preservation）",
		"对常规轮廓做封闭区域泛洪得到字腔；逐档加粗并复测面积/腐蚀深度，过粗方案被拒绝，采用最大安全粗度", fs)

	runes := []rune{'a', '国'}
	colW := 528.0
	gap := 24.0
	x0 := 48.0
	xs := []float64{x0, x0 + colW + gap, x0 + 2*(colW+gap)}
	top0 := 150.0
	rowH := 470.0
	rowGap := 26.0
	ppem := 250.0

	colTitles := []string{"常规 Regular", "自适应安全粗体（输出采用）", " naive 过粗（被校验拒绝）"}
	for ri, r := range runes {
		y0 := top0 + float64(ri)*(rowH+rowGap)
		for ci := 0; ci < 3; ci++ {
			cx0 := xs[ci]
			sc.Panel(cx0, y0, cx0+colW, y0+rowH, CPanel)
			switch ci {
			case 1:
				sc.FrameRect(cx0+2, y0+2, cx0+colW-2, y0+rowH-2, 3.5, CGreen)
			case 2:
				sc.FrameRect(cx0+2, y0+2, cx0+colW-2, y0+rowH-2, 3.5, CRed)
			}
			sc.DrawText(colTitles[ci], cx0+24, y0+42, fs,
				TextStyle{Ppem: 21, Color: CInk, Bold: true})
			sc.DrawText(fmt.Sprintf("字形 %c  U+%04X", r, r), cx0+colW-150, y0+42, fs,
				TextStyle{Ppem: 16, Color: CMuted})

			cx := cx0 + colW/2
			base := y0 + 300
			ink := fs.RuneInk(r, ppem, false)
			penX := cx - (ink.MinX+ink.MaxX)/2

			switch ci {
			case 0:
				sc.DrawRuneRadius(fs, r, penX, base, TextStyle{Ppem: ppem, Color: CInk}, 0)
				rep := fs.BoldReport(r)
				nC := len(rep.AtChosen)
				sc.DrawText(fmt.Sprintf("基准：%d 个封闭字腔", nC), cx0+24, y0+rowH-58, fs,
					TextStyle{Ppem: 18, Color: CInkSoft})
				sc.DrawText("填充原始 glyf 轮廓", cx0+24, y0+rowH-30, fs,
					TextStyle{Ppem: 15, Color: CMuted})
			case 1:
				rep := fs.BoldReport(r)
				rad := rep.ChosenEm * ppem
				sc.DrawRuneRadius(fs, r, penX, base, TextStyle{Ppem: ppem, Color: CInk}, rad)
				minRet, open := 1.0, 0
				for _, st := range rep.AtChosen {
					if st.Retention < minRet {
						minRet = st.Retention
					}
					if st.AreaBold >= 28 {
						open++
					}
				}
				badge := fmt.Sprintf("PASS · r=%.3f em · 字腔保留 %d/%d · 最小面积比 %.0f%%",
					rep.ChosenEm, open, len(rep.AtChosen), minRet*100)
				sc.FillRect(cx0+18, y0+rowH-66, cx0+colW-18, y0+rowH-24, CGreenSoft)
				sc.DrawText(badge, cx0+32, y0+rowH-38, fs,
					TextStyle{Ppem: 17, Color: CGreen, Bold: true})
			case 2:
				const badR = 0.11
				_, stats, _, _, _, _, _ := fs.EvaluateRadiusEm(r, badR)
				rad := badR * ppem
				sc.DrawRuneRadius(fs, r, penX, base, TextStyle{Ppem: ppem, Color: CInk}, rad)
				closed := 0
				for _, st := range stats {
					if st.AreaBold < 28 {
						closed++
					}
				}
				badge := fmt.Sprintf("FAIL · r=%.2f em · %d/%d 个字腔被吞",
					badR, closed, len(stats))
				sc.FillRect(cx0+18, y0+rowH-66, cx0+colW-18, y0+rowH-24, CRedSoft)
				sc.DrawText(badge, cx0+32, y0+rowH-38, fs,
					TextStyle{Ppem: 17, Color: CRed, Bold: true})
			}
		}
	}

	concl := "结论：所有正式产物只使用 PASS 粗度；密集汉字自动减小膨胀半径，字腔在任意字号下保持拓扑开放。"
	sc.FillRect(48, float64(H)-78, 1672, float64(H)-24, CBlueSoft)
	sc.DrawText(concl, 72, float64(H)-43, fs, TextStyle{Ppem: 19, Color: CBlue, Bold: true})
	return sc
}
