import { t } from '~util/i18n'

const MONTH_KEYS = Array.from({ length: 12 }, (_, i) => `util.date-format.month-${i + 1}`)
const DAY_KEYS = Array.from({ length: 7 }, (_, i) => `util.date-format.day-${i + 1}`)

function padString(paddingValue: string, str: string) {
  return String(paddingValue + str).slice(-paddingValue.length)
}

function formatTimeAgo(diffMs: number) {
  const minutes = Math.floor(diffMs / 1000 / 60)
  if (minutes < 1) {
    return t('util.date-format.just-now')
  }
  if (minutes < 60) {
    return t('util.date-format.minutes-ago', { minutes })
  }
  const hours = Math.floor(minutes / 60)
  if (hours < 24) {
    return t('util.date-format.hours-ago', { hours })
  }
  const days = Math.floor(hours / 24)
  return t('util.date-format.days-ago', { days })
}

export default function formatDate(date: Date, format: string = '%m.%d.%Y %H:%M') {
  const localizedMonthNames = MONTH_KEYS.map(key => t(key))
  const localizedDayNames = DAY_KEYS.map(key => t(key))

  if (!format) format = window.localStorage.dateFormat
  var months = localizedMonthNames
  var days = localizedDayNames
  if (!months || !days || !format) {
    return (
      padString('00', date.getDate().toString()) +
      '.' +
      padString('00', (date.getMonth() + 1).toString()) +
      '.' +
      padString('0000', date.getFullYear().toString()) +
      ' ' +
      padString('00', date.getHours().toString()) +
      ':' +
      padString('00', date.getMinutes().toString()) +
      ':' +
      padString('00', date.getSeconds().toString())
    )
  }

  var dayWeekU = (date.getDay() + 6) % 7
  var dayWeekW = date.getDay()

  var hour24 = date.getHours()
  var hour12 = date.getHours() % 12
  var isPM = hour24 > 12 // false = am
  if (!hour12) hour12 = 12

  // sorry for comments in russian, ported from another project, cba replacing.
  // this is just php's strftime replication. compatible with python too for the most part
  // %a = день недели строкой
  // %d = день месяца с нулями
  // %e = день месяца с пробелом
  // %u = день недели (1 = понедельник)
  // %w = день недели (0 = воскресенье)
  // %b, %h = месяц
  // %m = месяц с нулями
  // %C = век = Math.floor(year / 100)
  // %y = год, 2 символа
  // %Y = год, 4 символа
  // %H = час, 0-23, с нулями
  // %k = час, 0-23, с пробелом
  // %I = час, 1-12, с нулями
  // %l = час, 1-12, с пробелом
  // %M = минута, 0-59, с нулями
  // %p = AM/PM
  // %P = am/pm
  // %r = %I:%M:%S %p
  // %R = %H:%M
  // %S = секунда, 0-59, с нулями
  // %s = timestamp
  // %O = N дней назад

  var s = format
  s = s
    .replace(/%h/g, '%b')
    .replace(/%r/g, '%I:%M:%s %p')
    .replace(/%R/g, '%H:%M')
    .replace(/%a/g, days[dayWeekU])
    .replace(/%d/g, padString('00', date.getDate().toString()))
    .replace(/%e/g, padString('  ', date.getDate().toString()))
    .replace(/%u/g, String(dayWeekU + 1))
    .replace(/%w/g, String(dayWeekW))
    .replace(/%b/g, months[date.getMonth()])
    .replace(/%m/g, padString('00', (date.getMonth() + 1).toString()))
    .replace(/%C/g, String(Math.ceil(date.getFullYear() / 100)))
    .replace(/%y/g, padString('00', date.getFullYear().toString().substring(2)))
    .replace(/%Y/g, padString('0000', date.getFullYear().toString()))
    .replace(/%H/g, padString('00', date.getHours().toString()))
    .replace(/%k/g, padString('  ', date.getHours().toString()))
    .replace(/%I/g, padString('00', hour12.toString()))
    .replace(/%l/g, padString('  ', hour12.toString()))
    .replace(/%M/g, padString('00', date.getMinutes().toString()))
    .replace(/%p/g, isPM ? 'PM' : 'AM')
    .replace(/%P/g, isPM ? 'pm' : 'am')
    .replace(/%S/g, padString('00', date.getSeconds().toString()))
    .replace(/%s/g, String(Math.floor(date.getTime() / 1000)))
    .replace(/%O/g, formatTimeAgo(new Date().getTime() - date.getTime()))
  return s
}