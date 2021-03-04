const basicCustomEntryPoint = async function () {
  let str = ""

  try {
    const { v4: uuid } = require('uuid')
    str = uuid()
  } catch (error) {
    throw `Failed uuid string generation: ${error}`
  }

  if (str.length === 0) {
    throw `Failed uuid string generation: string empty`
  }

  return `Successfully created uuid string: ${str}`
}

exports.handler = async () => {
  return await basicCustomEntryPoint()
}
