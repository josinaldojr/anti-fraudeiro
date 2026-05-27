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

// func hasAVX2() bool
TEXT ·hasAVX2(SB), NOSPLIT, $0
    // Check max CPUID level
    MOVL $0, AX
    CPUID
    CMPL AX, $7
    JL no_avx2

    // Check OSXSAVE and AVX support first in EAX=1
    MOVL $1, AX
    CPUID
    // ECX bit 27 = OSXSAVE, bit 28 = AVX
    // Mask: (1 << 27) | (1 << 28) = 0x18000000
    ANDL $0x18000000, CX
    CMPL CX, $0x18000000
    JNE no_avx2

    // Check if OS enabled YMM/XMM saving
    MOVL $0, CX
    BYTE $0x0f; BYTE $0x01; BYTE $0xd0 // XGETBV instruction (since older go assemblers might not recognize XGETBV literally)
    // EAX bit 1 = XMM, bit 2 = YMM
    // Mask: 6
    ANDL $6, AX
    CMPL AX, $6
    JNE no_avx2

    // Check AVX2 in CPUID EAX=7, ECX=0
    MOVL $7, AX
    MOVL $0, CX
    CPUID
    // EBX bit 5 = AVX2 (1 << 5 = 32)
    ANDL $32, BX
    CMPL BX, $32
    JE has_avx2

no_avx2:
    MOVB $0, ret+0(FP)
    RET

has_avx2:
    MOVB $1, ret+0(FP)
    RET

