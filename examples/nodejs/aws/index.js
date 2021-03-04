const log = require('SyntheticsLogger')
const AWS = require('aws-sdk')

const basicCustomEntryPoint = async function () {
  log.info('Starting DynamoDB:listTables canary.')

  const dynamodb = new AWS.DynamoDB()
  const request = await dynamodb.listTables()
  try {
    const response = await request.promise()
    log.info('listTables response: ' + JSON.stringify(response))
  } catch (err) {
    log.error('listTables error: ' + JSON.stringify(err), err.stack)
    throw err
  }

  return 'Successfully completed DynamoDB:listTables canary.'
}

exports.handler = async () => {
  return await basicCustomEntryPoint()
}
