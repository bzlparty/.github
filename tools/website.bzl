load("@rules_pkg//pkg:providers.bzl", "PackageFilesInfo")
load("@rules_pkg//pkg:mappings.bzl", "pkg_filegroup", "pkg_files")

def root_index(name, src, pkgs):
  native.genrule(
    name = "%s_index" % name,
    srcs = [src],
    outs = ["index.md"],
    cmd = """
(
sed "1 s|(.*)|(/assets/logo.jpg)|" $(location {src});
echo "\n## Party stuff";
for r in {rules}; do
  echo "- [$$r]($$r)"
done;
) > $(OUTS)""".format(rules = " ".join(pkgs), src = src),
  )

  pkg_files(
    name = name,
    srcs = ["%s_index" % name],
  )

def _generate_html_impl(ctx):
  outputs = []
  args = ctx.actions.args()
  fileargs = []
  asset_args = []
  dsm = dict()

  for p in ctx.attr.srcs + ctx.attr.assets:
    dest_src_map = getattr(p[PackageFilesInfo], "dest_src_map")

    for d, s in dest_src_map.items():
      if d.endswith(".md"):
        t = d.replace(".md", ".html")
        output = ctx.actions.declare_file(t)
        outputs.append(output)
        fileargs.append("%s:%s" % (s.path, output.path))
        dsm.update([(t, output)])

      if d.endswith(".css"):
        asset_args.append("/%s" % d)

  args.add_joined("-files", fileargs, join_with = ",")

  args.add_all("-template", [ctx.file.template])
  args.add_joined("-assets", asset_args, join_with = ",")
  args.add_all("-exclude-path", [ctx.bin_dir.path])

  args.add_all("-package", [ctx.attr.package])

  ctx.actions.run(
    inputs = ctx.files.srcs + [ctx.file.template],
    outputs = outputs,
    arguments = [args],
    mnemonic = "Builder",
    progress_message = "Build %s" % ctx.attr.name,
    executable = ctx.executable.builder,
  )

  return [
    DefaultInfo(files = depset(outputs)),
    PackageFilesInfo(dest_src_map = dsm, attributes = {}),
  ]

html_files = rule(
  _generate_html_impl,
  attrs = {
    "srcs": attr.label_list(
      providers = [PackageFilesInfo],
    ),
    "package": attr.string(),
    "assets": attr.label_list(),
    "template": attr.label(
      default = "//tools:page.tpl.html",
      allow_single_file = True,
    ),
    "builder": attr.label(
      default = "//tools:builder",
      cfg = "exec",
      executable = True,
    )
  }
)

def html_filegroup(name, srcs, assets, prefix):
  for s in srcs:
    html_files(
      name = "%s_html" % s,
      package = s if s != "root" else None,
      srcs = [s],
      assets = [assets],
    )

  pkg_filegroup(
    name = name,
    srcs = ["%s_html" % s for s in srcs] + [assets],
    prefix = prefix,
  )
