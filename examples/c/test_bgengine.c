// Test program for libbgengine shared library
// Compile: gcc -o test_bgengine test_bgengine.c -L../.. -lbgengine -Wl,-rpath,'$ORIGIN/../..'
// Run: ./test_bgengine

#include <stdio.h>
#include <stdlib.h>
#include "../../libbgengine.h"

int main() {
    printf("=== GoBG C Library Test ===\n\n");

    // Get version
    char* version = bgengine_version();
    printf("Library version: %s\n\n", version);
    bgengine_free_string(version);

    // Initialize engine
    printf("Initializing engine...\n");
    int result = bgengine_init(
        "data/gnubg.weights",
        "data/gnubg_os0.bd",
        "data/gnubg_ts.bd",
        "data/g11.xml"
    );
    
    if (result != 0) {
        char* error = bgengine_last_error();
        printf("Failed to initialize engine: %s\n", error ? error : "unknown error");
        if (error) bgengine_free_string(error);
        return 1;
    }
    printf("Engine initialized successfully!\n\n");

    // Test evaluate
    printf("--- Evaluate Starting Position ---\n");
    char* json = NULL;
    result = bgengine_evaluate("4HPwATDgc/ABMA", &json);
    if (result == 0) {
        printf("Result: %s\n", json);
    } else {
        printf("Error: %s\n", json);
    }
    bgengine_free_string(json);
    printf("\n");

    // Test best move
    printf("--- Best Move (3-1) ---\n");
    result = bgengine_best_move("4HPwATDgc/ABMA", 3, 1, &json);
    if (result == 0) {
        printf("Result: %s\n", json);
    } else {
        printf("Error: %s\n", json);
    }
    bgengine_free_string(json);
    printf("\n");

    // Test cube decision
    printf("--- Cube Decision ---\n");
    result = bgengine_cube_decision("4HPwATDgc/ABMA", &json);
    if (result == 0) {
        printf("Result: %s\n", json);
    } else {
        printf("Error: %s\n", json);
    }
    bgengine_free_string(json);
    printf("\n");

    // Shutdown
    bgengine_shutdown();
    printf("Engine shutdown complete.\n");

    return 0;
}

