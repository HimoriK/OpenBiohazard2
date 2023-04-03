package render

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/OpenBiohazard2/OpenBiohazard2/fileio"
	"github.com/OpenBiohazard2/OpenBiohazard2/game"
	"github.com/OpenBiohazard2/OpenBiohazard2/geometry"
)

const (
	RENDER_TYPE_ENTITY = 3
	VERTEX_LEN         = 8
)

var (
	animation = NewAnimation()
)

type PlayerEntity struct {
	TextureId           uint32
	VertexBuffer        []float32
	PLDOutput           *fileio.PLDOutput
	Player              *game.Player
	AnimationPoseNumber int
	VertexArrayObject   uint32
	VertexBufferObject  uint32
}

// Offset in vertex buffer
type ComponentOffsets struct {
	StartIndex int
	EndIndex   int
}

func NewPlayerEntity(pldOutput *fileio.PLDOutput) *PlayerEntity {
	// Generate buffers
	var vao uint32
	gl.GenVertexArrays(1, &vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)

	textureId := NewTextureTIM(pldOutput.TextureData)
	vertexBuffer := geometry.NewMD1Geometry(pldOutput.MeshData, pldOutput.TextureData)

	return &PlayerEntity{
		TextureId:           textureId,
		VertexBuffer:        vertexBuffer,
		PLDOutput:           pldOutput,
		Player:              nil,
		AnimationPoseNumber: -1,
		VertexArrayObject:   vao,
		VertexBufferObject:  vbo,
	}
}

func (playerEntity *PlayerEntity) UpdatePlayerEntity(player *game.Player, animationPoseNumber int) {
	playerEntity.Player = player
	playerEntity.AnimationPoseNumber = animationPoseNumber
}

func RenderAnimatedEntity(programShader uint32, playerEntity PlayerEntity, timeElapsedSeconds float64) {
	texId := playerEntity.TextureId
	pldOutput := playerEntity.PLDOutput
	entityVertexBuffer := playerEntity.VertexBuffer

	renderTypeUniform := gl.GetUniformLocation(programShader, gl.Str("renderType\x00"))
	gl.Uniform1i(renderTypeUniform, RENDER_TYPE_ENTITY)

	modelLoc := gl.GetUniformLocation(programShader, gl.Str("model\x00"))
	modelMatrix := playerEntity.Player.GetModelMatrix()
	gl.UniformMatrix4fv(modelLoc, 1, false, &modelMatrix[0])

	animation.UpdateAnimationFrame(playerEntity.AnimationPoseNumber, playerEntity.PLDOutput.AnimationData, timeElapsedSeconds)

	// The root of the skeleton is component 0
	transforms := make([]mgl32.Mat4, len(pldOutput.MeshData.Components))
	buildComponentTransforms(pldOutput.SkeletonData, 0, -1, transforms)

	// Build vertex and texture data
	componentOffsets := calculateComponentOffsets(pldOutput.MeshData)
	floatSize := 4

	// 3 floats for vertex, 2 floats for texture UV, 3 float for normals
	stride := int32(VERTEX_LEN * floatSize)

	vao := playerEntity.VertexArrayObject
	gl.BindVertexArray(vao)

	vbo := playerEntity.VertexBufferObject
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(entityVertexBuffer)*floatSize, gl.Ptr(entityVertexBuffer), gl.STATIC_DRAW)

	// Position attribute
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// Texture
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*floatSize))
	gl.EnableVertexAttribArray(1)

	// Normal
	gl.VertexAttribPointer(2, 3, gl.FLOAT, false, stride, gl.PtrOffset(5*floatSize))
	gl.EnableVertexAttribArray(2)

	diffuseUniform := gl.GetUniformLocation(programShader, gl.Str("diffuse\x00"))
	gl.Uniform1i(diffuseUniform, 0)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texId)

	// Draw triangles
	for i := 0; i < len(componentOffsets); i++ {
		// Set offset to translate each component relative to model origin
		boneOffsetLoc := gl.GetUniformLocation(programShader, gl.Str("boneOffset\x00"))
		boneOffset := transforms[i]
		gl.UniformMatrix4fv(boneOffsetLoc, 1, false, &boneOffset[0])

		startIndex := componentOffsets[i].StartIndex
		endIndex := componentOffsets[i].EndIndex

		// Render model component
		vertOffset := int32(startIndex / VERTEX_LEN)
		numVertices := int32((endIndex - startIndex) / VERTEX_LEN)
		gl.DrawArrays(gl.TRIANGLES, vertOffset, numVertices)
	}

	// Cleanup
	gl.DisableVertexAttribArray(0)
	gl.DisableVertexAttribArray(1)
	gl.DisableVertexAttribArray(2)
}

func buildComponentTransforms(skeletonData *fileio.EMROutput, curId int, parentId int, transforms []mgl32.Mat4) {
	transformMatrix := mgl32.Ident4()
	if parentId != -1 {
		transformMatrix = transforms[parentId]
	}

	offsetFromParent := skeletonData.RelativePositionData[curId]

	// Translate from parent offset
	translate := mgl32.Translate3D(float32(offsetFromParent.X), float32(offsetFromParent.Y), float32(offsetFromParent.Z))
	transformMatrix = transformMatrix.Mul4(translate)

	// Rotate if there is an animation pose
	if animation.CurPose != -1 {
		quat := mgl32.QuatIdent()
		frameRotation := skeletonData.FrameData[animation.FrameNumber].RotationAngles[curId]
		quat = quat.Mul(mgl32.QuatRotate(frameRotation.X(), mgl32.Vec3{1.0, 0.0, 0.0}))
		quat = quat.Mul(mgl32.QuatRotate(frameRotation.Y(), mgl32.Vec3{0.0, 1.0, 0.0}))
		quat = quat.Mul(mgl32.QuatRotate(frameRotation.Z(), mgl32.Vec3{0.0, 0.0, 1.0}))
		transformMatrix = transformMatrix.Mul4(quat.Mat4())
	}

	transforms[curId] = transformMatrix

	for i := 0; i < len(skeletonData.ArmatureChildren[curId]); i++ {
		newParent := curId
		newChild := int(skeletonData.ArmatureChildren[curId][i])
		buildComponentTransforms(skeletonData, newChild, newParent, transforms)
	}
}

func calculateComponentOffsets(meshData *fileio.MD1Output) []ComponentOffsets {
	componentOffsets := make([]ComponentOffsets, len(meshData.Components))
	startIndex := 0
	endIndex := 0
	for i, entityModel := range meshData.Components {
		startIndex = endIndex
		triangleBufferCount := len(entityModel.TriangleIndices) * 3 * VERTEX_LEN
		quadBufferCount := len(entityModel.QuadIndices) * 3 * 2 * VERTEX_LEN
		endIndex = startIndex + (triangleBufferCount + quadBufferCount)

		componentOffsets[i] = ComponentOffsets{
			StartIndex: startIndex,
			EndIndex:   endIndex,
		}
	}
	return componentOffsets
}
