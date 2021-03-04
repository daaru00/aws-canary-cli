const log = require('SyntheticsLogger')
const AWS = require('aws-sdk')

const basicCustomEntryPoint = async function () {
  log.info('Starting SSM:GetParametersByPath canary.')

  const ssm = new AWS.SSM()
  const params = {
    Path: '/cwsyn/',
    Recursive: true
  }
  const request = await ssm.getParametersByPath(params)
  try {
    const response = await request.promise()
    log.info('getParametersByPath response: ' + JSON.stringify(response))
  } catch (err) {
    log.error('getParametersByPath error: ' + JSON.stringify(err), err.stack)
    throw err
  }

  return 'Successfully completed SSM:GetParametersByPath canary.'
}

exports.handler = async () => {
  return await basicCustomEntryPoint()
}
