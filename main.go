package main

import (
	"log"
	"math"
	"math/rand"

	"github.com/MatiasLyyra/fluid/simulation"
	rl "github.com/gen2brain/raylib-go/raylib"
)

const MaxNeighbours = 32
const ParticleCount = 1 << 14
const Width = 768
const Height = 768
const Radius = 0.04

const NoNeighbour = math.MaxUint32

type NeighbourList [ParticleCount * MaxNeighbours]int

const vertexShader = `
#version 430
in vec2 vertexPosition;
in vec2 instancePosition;
in uint code;
out vec2 fragPos;
uniform mat4 mvp;

void main()
{
	vec2 pos = vertexPosition * 4 + instancePosition;
	gl_Position = mvp * vec4(pos, 0.0, 1.0);
	fragPos = pos;
}
`

const fragmentShader = `
#version 430
in vec2 fragPos;
out vec4 fragColor;

void main()
{
	vec2 center = fragPos - 4;
	if (length(gl_FragCoord.xy - fragPos) > 4) discard;
	fragColor = vec4(1.0);
}
`

func HashPosition(pos, rangeLow, rangeHigh simulation.Vector2, step float32) uint32 {
	pos = pos.Clamp(rangeLow, rangeHigh)
	gridX := uint32((pos.X - rangeLow.X) / step)
	gridY := uint32((pos.Y - rangeLow.Y) / step)
	return interleave2d(gridX) | interleave2d(gridY)<<1
}

func interleave2d(v uint32) uint32 {
	v = (v ^ (v << 8)) & 0x00ff00ff
	v = (v ^ (v << 4)) & 0x0f0f0f0f
	v = (v ^ (v << 2)) & 0x33333333
	return (v ^ (v << 1)) & 0x55555555
}

func main() {

	particles := make([]simulation.Particle, ParticleCount)

	for i := range particles {
		particles[i] = simulation.Particle{
			Position: simulation.Vector2{
				X: rand.Float32(),
				Y: rand.Float32(),
			},
		}
		// particles[i].Code = HashPosition(particles[i].Position, simulation.Vector2{X: 0, Y: 0}, simulation.Vector2{X: 1, Y: 1}, 0.02)
	}
	// slices.SortFunc(particles, func(a, b simulation.Particle) int { return cmp.Compare(a.Code, b.Code) })
	rl.InitWindow(Width, Height, "Fluid")
	rl.SetTargetFPS(60)

	shader := rl.LoadShaderFromMemory(vertexShader, fragmentShader)
	// model := rl.LoadModelFromMesh(rl.GenMeshPlane(1, 1, 1, 1))
	// // vertexBuffer := rl.LoadVertexArray()
	// gl.BufferData(gl.ARRAY_BUFFER, len(particles)*int(unsafe.Sizeof(simulation.Particle{})), unsafe.Pointer(unsafe.SliceData(particles)), gl.STATIC_DRAW)

	// gl.BindVertexArray(model.Meshes.VaoID)

	log.Printf("Data: %+v", rl.GetShaderLocationAttrib(shader, "fragPos"))
}

func smoothingKernel(rSq, h float32) float32 {
	norm := math.Pi * float32(math.Pow(float64(h), 8)) / 4
	diff := max(h*h-rSq, 0)
	return norm * diff * diff * diff
}
