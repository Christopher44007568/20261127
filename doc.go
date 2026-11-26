// Package stroke tessellates wide 2D polylines into flat triangles and renders
// the result as a solid fill, antialiased fill, or overlap diagnostic.
//
// It supports miter, round, and bevel joins; butt, round, and square caps;
// arc-length dashing; zero-length subpaths; miter limiting; and affine
// transforms with explicit width policies. It uses only the Go standard
// library.
package stroke
