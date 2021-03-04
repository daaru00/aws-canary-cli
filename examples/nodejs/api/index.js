const log = require('SyntheticsLogger')
const https = require('https')
const url = new URL(process.env.ENDPOINT)

const request = async (options, data) => {
  options = options || {}
  
  return new Promise((resolve, reject) => {
    const req = https.request({
      host: url.host,
      username: url.username,
      password: url.password,
      path: url.pathname,
      search: url.search,
      ...options
    }, (res) => {
      let data = '';
      res.on('data', function (chunk) {
        data += chunk;
      });
      res.on('error', function (err) {
        reject(err)
      });
      res.on('timeout', function () {
        reject(new Error('RequestTimeout'))
      });
      res.on('end', function () {
        resolve({
          headers: res.headers,
          statusCode: res.statusCode,
          data
        });
      });
    })
    
    if (data) {
      req.write(data);
    }

    req.setTimeout(parseInt(process.env.RESPONSE_TIMEOUT))
    req.end();
  })
}

const basicCustomEntryPoint = async function () {
  try {
    let res = await request({
      headers: {
        'app-id': process.env.API_KEY,
        'x-api-key': process.env.API_KEY
      }
    })
    log.info('API response: ' + JSON.stringify(res))
    if (res.statusCode !== 200) {
      throw res.data
    }
  } catch (err) {
    log.error('API error: ' + JSON.stringify(err), err.stack)
    throw err
  }

  return `Successfully completed ${process.env.ENDPOINT} API checks.`
}

exports.handler = async () => {
  return await basicCustomEntryPoint()
}
