const basicCustomEntryPoint = async function () {
  let fail = false
  if (fail) {
    throw `Failed ${process.env.TEST_NAME} check.`
  }

  return `Successfully completed ${process.env.TEST_NAME} checks.`
}

exports.handler = async () => {
  return await basicCustomEntryPoint()
}
