.global _start
.align 2

_start:
    // Simple register operations
    mov x0, #42
    mov x1, #7
    add x2, x0, x1
    sub x3, x0, x1
    
    // Exit syscall
    mov x0, #0      // Exit code
    mov x16, #1     // Exit syscall number for macOS
    svc #0x80       // Trigger syscall
