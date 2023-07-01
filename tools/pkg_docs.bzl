load("@rules_pkg//pkg:mappings.bzl", "pkg_files")

def docs_files(pkgs):
  for p in pkgs:
    pkg_files(
      name = p,
      srcs = ["@%s//:docs" % p],
      renames = {
        "@%s//:README.md" % p: "index.md",
      },
      prefix = p,
    )
