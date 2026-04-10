def feature_paths(name):
    feature_dir = format("examples/poem/docs/features/{name}", name = name)
    return {
        "feature_dir": feature_dir,
        "spec_path": format("{dir}/spec.md", dir = feature_dir),
        "poem_path": format("{dir}/poem.md", dir = feature_dir),
        "review_path": format(
            "{dir}/review-{iter}.txt",
            dir = feature_dir,
            iter = loop_iter("extend_until_ready"),
        ),
    }
