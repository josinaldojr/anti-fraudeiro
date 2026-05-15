// +build amd64,!noasm

#include "textflag.h"

// func DistancesAVX2(query *[16]int32, vectors []uint16, distances []uint64)
TEXT ·DistancesAVX2(SB), NOSPLIT, $0
    // Load query pointer
    MOVQ query+0(FP), SI
    // Load vectors pointer and length
    MOVQ vectors+8(FP), DI
    MOVQ vectors+16(FP), DX // length in uint16
    // Load distances pointer
    MOVQ distances+32(FP), R8

    // Load query into YMM0, YMM1 (16 * int32)
    VMOVDQU (SI), Y0
    VMOVDQU 32(SI), Y1

    // Number of vectors = length / 16
    SHRQ $4, DX
    JZ done

loop:
    // Load 8 uint16 from vectors, extend to int32 in YMM2
    VPMOVZXWD (DI), Y2
    // Load next 8 uint16 from vectors, extend to int32 in YMM3
    VPMOVZXWD 16(DI), Y3

    // Subtract from query: Y4 = Y0 - Y2, Y5 = Y1 - Y3
    VPSUBD Y2, Y0, Y4
    VPSUBD Y3, Y1, Y5

    // Square: Y4 = Y4 * Y4, Y5 = Y5 * Y5
    // VPMULLD is fine because 65535^2 < 2^32
    VPMULLD Y4, Y4, Y4
    VPMULLD Y5, Y5, Y5

    // Sum squares: Y6 = Y4 + Y5 (8x int32)
    VPADDD Y4, Y5, Y6

    // Now we have 8x 32-bit sums. We need to sum them into a 64-bit result.
    // To avoid overflow, extend to 64-bit and sum.
    VPMOVZXDQ X6, Y7       // Low 4 to Y7 (4x 64-bit)
    VEXTRACTI128 $1, Y6, X8
    VPMOVZXDQ X8, Y9       // High 4 to Y9 (4x 64-bit)
    VPADDQ Y7, Y9, Y10      // Sum (4x 64-bit)

    // Horizontal sum of Y10 (4x 64-bit)
    VEXTRACTI128 $1, Y10, X11
    VPADDQ X10, X11, X12    // Sum (2x 64-bit)
    VPSHUFD $0x4E, X12, X13
    VPADDQ X12, X13, X14    // Sum (1x 64-bit in low 64 bits)

    // Store result
    VMOVQ X14, (R8)

    // Advance pointers
    ADDQ $32, DI
    ADDQ $8, R8
    DECQ DX
    JNZ loop

done:
    VZEROUPPER
    RET
