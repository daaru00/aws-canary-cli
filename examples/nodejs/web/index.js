var synthetics = require('Synthetics')
const log = require('SyntheticsLogger')

const pageLoadBlueprint = async function () {
  let page = await synthetics.getPage()

  const response = await page.goto(process.env.ENDPOINT, {
    waitUntil: 'domcontentloaded',
    timeout: process.env.PAGE_LOAD_TIMEOUT
  })
  if (!response) {
    throw 'Failed to load page!'
  }

  await page.waitFor(15000)
  await synthetics.takeScreenshot('loaded', 'loaded')

  let pageTitle = await page.title()
  log.info('Page title: ' + pageTitle)

  if (response.status() < 200 || response.status() > 299) {
    throw 'Failed to load page!'
  }
}

exports.handler = async () => {
  return await pageLoadBlueprint()
}
