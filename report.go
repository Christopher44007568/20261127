package main

import (
	"fmt"
	"os"
	"strings"
)

// WriteSummary writes a plain-text verification report (Chinese, UTF-8).
func WriteSummary(path string, cfg GaitConfig, s Summary) error {
	var b strings.Builder
	fmt.Fprintln(&b, "斜坡双腿行走 - 两段 IK / 关节限位 / 脚掌锁地 验证摘要")
	fmt.Fprintln(&b, "====================================================")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "斜坡倾角: %.0f 度\n", radToDeg(cfg.Slope.Ang))
	fmt.Fprintf(&b, "大腿 L1=%.2f m, 小腿 L2=%.2f m, 最大可达 R=%.2f m\n", cfg.Leg.L1, cfg.Leg.L2, cfg.Leg.Reach())
	fmt.Fprintf(&b, "屈膝极限对应最近距离 FlexFloor=%.3f m\n", cfg.Leg.FlexFloor())
	fmt.Fprintf(&b, "关节限位(度): 髋[%.0f, %.0f] 膝[%.0f, %.0f] 踝[%.0f, %.0f]\n",
		radToDeg(cfg.Leg.HipLimit.Min), radToDeg(cfg.Leg.HipLimit.Max),
		radToDeg(cfg.Leg.KneeLimit.Min), radToDeg(cfg.Leg.KneeLimit.Max),
		radToDeg(cfg.Leg.AnkleLimit.Min), radToDeg(cfg.Leg.AnkleLimit.Max))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "密集采样足端样本数: %d\n", s.Samples)
	fmt.Fprintf(&b, "行走轨迹最大 IK 末端误差: %.3e m\n", s.MaxIKError)
	fmt.Fprintf(&b, "支撑期脚掌锁地最大漂移: %.3e m\n", s.MaxLockDrift)
	fmt.Fprintf(&b, "髋角范围: [%.2f, %.2f] 度\n", s.HipMin, s.HipMax)
	fmt.Fprintf(&b, "膝屈范围: [%.2f, %.2f] 度  (最小值>=0 即无膝反折)\n", s.KneeMin, s.KneeMax)
	fmt.Fprintf(&b, "踝偏范围: [%.2f, %.2f] 度\n", s.AnkleMin, s.AnkleMax)
	fmt.Fprintf(&b, "行走中超可达帧: %d, 限位钳制帧: %d (主步态应全部为 0)\n", s.OverReachFrames, s.LimitFrames)
	fmt.Fprintf(&b, "可达扫描中超可达段最大误差: %.3f m (= Dmax - R)\n", s.SweepMaxOverErr)
	fmt.Fprintln(&b)

	checks := []struct {
		name string
		ok   bool
		hint string
	}{
		{"行走 IK 误差 < 1e-9 m", s.MaxIKError < 1e-9, "两段解析解在可达环内应为机器精度"},
		{"脚掌锁地漂移 < 1e-9 m", s.MaxLockDrift < 1e-9, "支撑期足底坐标不随时间变化"},
		{"膝盖无反折 (KneeMin >= -1e-9)", s.KneeMin >= -1e-9, "极点选支 + 角度钳制保证"},
		{"膝盖未超最大屈曲", s.KneeMax <= radToDeg(cfg.Leg.KneeLimit.Max)+1e-6, ""},
	}
	fmt.Fprintln(&b, "自检结论:")
	allOK := true
	for _, c := range checks {
		mark := "PASS"
		if !c.ok {
			mark = "FAIL"
			allOK = false
		}
		fmt.Fprintf(&b, "  [%s] %s\n", mark, c.name)
		if !c.ok && c.hint != "" {
			fmt.Fprintf(&b, "         -> %s\n", c.hint)
		}
	}
	if allOK {
		fmt.Fprintln(&b, "\n全部不变量通过。超可达目标的伸直+红色标记录于拼图第 4 行第 3 格与误差图下半部。")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func radToDeg(r float64) float64 { return r * 180 / 3.141592653589793 }

// WriteHTML writes a small Chinese gallery tying all artifacts together.
func WriteHTML(path string) error {
	html := `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8">
<title>斜坡双腿行走：两段 IK / 关节限位 / 脚掌锁地</title>
<style>
 body{font-family:-apple-system,"Noto Sans CJK SC","Microsoft YaHei",sans-serif;margin:32px;color:#1f2937;background:#f8fafc;line-height:1.7}
 h1{font-size:26px} h2{font-size:20px;margin-top:36px}
 img{max-width:100%;border:1px solid #d1d5db;border-radius:10px;background:#fff}
 code{background:#eef2ff;padding:2px 6px;border-radius:5px}
 .ok{color:#16a34a;font-weight:700}.bad{color:#dc2626;font-weight:700}
 li{margin:4px 0}
</style></head><body>
<h1>斜坡双腿行走：两段 IK · 关节限位 · 脚掌锁地</h1>
<p>纯 Go 标准库实现（无第三方包）。运行 <code>go run .</code> 重新生成全部产物，
<code>go test ./...</code> 验证不变量。</p>
<ul>
 <li>两段解析 IK：余弦定理求膝点，按“朝坡上/前进方向”的极点向量选择唯一膝支，<span class="ok">不会膝反折</span>。</li>
 <li>关节限位：髋 / 膝 / 踝角度区间，距离先钳入可行环，再做角度钳制 + 前向运动学。</li>
 <li>脚掌锁地：支撑期足底对应固定的沿坡坐标，世界坐标不随帧变化（漂移为机器精度）。</li>
 <li>超可达目标 <code>D &gt; L1+L2</code>：整条腿完全伸直指向目标、端点钳在 R 处，<span class="bad">红色</span>标出并给出误差。</li>
</ul>
<h2>行走帧拼图（12 个步态相位 + 4 个 IK 边界面板）</h2>
<img src="walk_frames_sheet.png" alt="walk frames sheet">
<h2>无缝循环动画</h2>
<img src="walk.gif" alt="walk cycle gif" style="max-width:520px">
<h2>目标与末端误差</h2>
<img src="target_end_error.png" alt="target vs end effector error">
<h2>关节角曲线</h2>
<img src="joint_angle_curves.png" alt="joint angle curves">
<h2>数据</h2>
<ul>
 <li><a href="metrics.csv">metrics.csv</a>：逐帧双足目标点、末端点、误差、锁地漂移与关节角。</li>
 <li><a href="reach_sweep.csv">reach_sweep.csv</a>：指令距离扫描的末端误差 / 膝角 / 状态。</li>
 <li><a href="summary.txt">summary.txt</a>：量化摘要与自检结论。</li>
</ul>
</body></html>
`
	return os.WriteFile(path, []byte(html), 0o644)
}
