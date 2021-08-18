# Wait for Android emulator

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-wait-for-android-emulator?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-wait-for-android-emulator/releases)

Wait for the emulator to finish boot

<details>
<summary>Description</summary>

If your workflow contains the `start-android-emulator` step,
and you've set the `wait_for_boot` parameter to `false`, use this step to check
if the android emulator is booted or wait for it to finish booting.
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://devcenter.bitrise.io/steps-and-workflows/steps-and-workflows-index/).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `emulator_serial` | Emulator with the given serial will be checked if booted, or wait for it to boot.  | required | `$BITRISE_EMULATOR_SERIAL` |
| `boot_timeout` | Maximum time to wait for emulator to boot.  | required | `300` |
| `android_home` | Android sdk path | required | `$ANDROID_HOME` |
</details>

<details>
<summary>Outputs</summary>
There are no outputs defined in this step
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-wait-for-android-emulator/pulls) and [issues](https://github.com/bitrise-steplib/steps-wait-for-android-emulator/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://devcenter.bitrise.io/bitrise-cli/run-your-first-build/).

Learn more about developing steps:

- [Create your own step](https://devcenter.bitrise.io/contributors/create-your-own-step/)
- [Testing your Step](https://devcenter.bitrise.io/contributors/testing-and-versioning-your-steps/)
