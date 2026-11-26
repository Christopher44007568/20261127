package main

import "fmt"

func passCheck(name string, pass bool, value interface{}, limit, detail string) Check {
	status := "PASS"
	if !pass {
		status = "FAIL"
	}
	return Check{Name: name + " [" + status + "]", Pass: pass, Value: value, Limit: limit, Detail: detail}
}

func buildDemoChecks(r *RunReport, geom GeometryScore, holes []HoleScore, components int, largest float64) []Check {
	checks := []Check{
		passCheck("outlier_recall", r.OutlierRecall >= 0.90, r.OutlierRecall, ">=0.90",
			fmt.Sprintf("%d/%d injected outliers rejected", r.RemovedOutliers, r.InjectedOutliers)),
		passCheck("surface_points_preserved", r.SurfaceFalseRemoved <= 20, r.SurfaceFalseRemoved,
			"<=20 false removals", "robust threshold protects one-sided hole boundaries"),
		passCheck("normals_oriented", r.Orientation.Flips > 1000 && r.Orientation.Components >= 1,
			r.Orientation, "flips>1000", "MST propagates one sign across the shell"),
		passCheck("dense_mesh", lenMeshTriangles(r) >= 10000, lenMeshTriangles(r), ">=10000", "implicit zero set extraction"),
		passCheck("one_main_component", largest >= 0.98, largest, ">=0.98",
			fmt.Sprintf("%d components after small-island cleanup", components)),
		passCheck("open_boundaries", r.Mesh["boundary_edges"].(int) >= 250,
			r.Mesh["boundary_edges"], ">=250", "outer patch and hole rims remain open"),
		passCheck("body_rmse", geom.RMSE < 0.008, geom.RMSE, "<0.008 world units",
			"normal-distance error against analytic shell"),
		passCheck("no_systematic_swelling", absFloat(geom.SignedMean) < 0.004,
			geom.SignedMean, "|mean signed|<0.004", "near-zero bias rules out oil-film dilation"),
	}

	allCoresOpen := true
	maxInset := 0.0
	totalCoreHits := 0
	for _, h := range holes {
		if h.CoreTriangleHits != 0 {
			allCoresOpen = false
		}
		totalCoreHits += h.CoreTriangleHits
		if h.DeepestInset > maxInset {
			maxInset = h.DeepestInset
		}
	}
	checks = append(checks,
		passCheck("hole_cores_not_bridged", allCoresOpen, totalCoreHits, "0 core triangles",
			"compact support + centroid density trim"),
		passCheck("hole_edge_band_limited", maxInset < 0.070, maxInset, "<0.070",
			"only a resolution-scale rim band may be trimmed inward"),
	)
	if r.Curvature != nil {
		checks = append(checks, passCheck("curvature_error_controlled", r.Curvature.RMSE < 0.20,
			r.Curvature.RMSE, "<0.20 1/world unit", "interior mean-curvature error at analysis scale"))
	}
	return checks
}

func buildGenericChecks(r *RunReport, components int, largest float64) []Check {
	return []Check{
		passCheck("outliers_rejected", r.RemovedOutliers >= 0, r.RemovedOutliers, "reported", "robust kNN distance filter"),
		passCheck("mesh_extracted", lenMeshTriangles(r) >= 100, lenMeshTriangles(r), ">=100", "implicit zero set extraction"),
		passCheck("main_component", largest >= 0.90, largest, ">=0.90",
			fmt.Sprintf("%d components", components)),
	}
}

func lenMeshTriangles(r *RunReport) int {
	if r == nil || r.Mesh == nil {
		return 0
	}
	if n, ok := r.Mesh["triangles"].(int); ok {
		return n
	}
	return 0
}
