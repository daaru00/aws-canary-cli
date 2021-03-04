import os
from aws_synthetics.selenium import synthetics_webdriver as webdriver

def basic_selenium_script():
    browser = webdriver.Chrome()
    browser.get(os.environ.get('ENDPOINT'))
    browser.save_screenshot('loaded.png')

def handler(event, context):
    basic_selenium_script()
