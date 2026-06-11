const std = @import("std");

pub fn build(b: *std.Build) void {
    const exe = b.addExecutable(.{
        .name = "zig-example",
        .root_module = b.createModule(.{
            .root_source_file = b.path("main.zig"),
            .target = b.resolveTargetQuery(.{}),
            .optimize = .ReleaseSafe,
        }),
    });
    b.installArtifact(exe);
}
