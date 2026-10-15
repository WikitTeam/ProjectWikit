import * as React from 'react'
import styled from 'styled-components'
import { ArticleFormField } from '../api/articles'

interface Props {
  fields: ArticleFormField[]
  values: Record<string, string>
  disabled?: boolean
  onChange: (name: string, value: string) => void
}

const Styles = styled.table`
  width: 100%;
  max-width: 60em;

  td {
    vertical-align: top;
  }

  td.form-label {
    width: 20%;
    padding-top: 0.4em;
    font-weight: bold;
  }

  input[type='text'],
  input[type='password'],
  textarea,
  select {
    width: 95%;
    box-sizing: border-box;
  }

  .form-hint {
    margin-top: 0.2em;
    font-size: 87%;
    color: #666;
  }

  .form-radio {
    display: inline-block;
    margin-right: 1em;
    white-space: nowrap;
  }

  @media (max-width: 767px) {
    tr,
    td,
    td.form-label {
      display: block;
      width: auto;
    }

    input[type='text'],
    input[type='password'],
    textarea,
    select {
      width: 100%;
    }
  }
`

export function isChecked(value?: string) {
  return ['1', 'true', 'yes', 'on'].includes((value ?? '').trim().toLowerCase())
}

export function formInitialValues(fields: ArticleFormField[], stored: Record<string, string>) {
  const values = { ...stored }
  for (const field of fields) {
    if (!(field.name in values)) {
      values[field.name] = field.default ?? ''
    }
  }
  return values
}

const ArticleFormFields: React.FC<Props> = ({ fields, values, disabled, onChange }) => {
  const control = (field: ArticleFormField) => {
    const value = values[field.name] ?? ''
    const id = `form-field-${field.name}`
    switch (field.type) {
      case 'static':
        return <div className="form-static">{field.value}</div>
      case 'checkbox':
        return (
          <input id={id} type="checkbox" checked={isChecked(value)} disabled={disabled} onChange={e => onChange(field.name, e.target.checked ? '1' : '0')} />
        )
      case 'select':
        if (field.options.length <= 4) {
          return (
            <div>
              {field.options.map(option => (
                <label key={option.key} className="form-radio">
                  <input
                    type="radio"
                    name={id}
                    value={option.key}
                    checked={value === option.key}
                    disabled={disabled}
                    onChange={() => onChange(field.name, option.key)}
                  />{' '}
                  {option.label}
                </label>
              ))}
            </div>
          )
        }
        return (
          <select id={id} value={value} disabled={disabled} onChange={e => onChange(field.name, e.target.value)}>
            {!field.options.some(option => option.key === value) && <option value={value}>{value}</option>}
            {field.options.map(option => (
              <option key={option.key} value={option.key}>
                {option.label}
              </option>
            ))}
          </select>
        )
      case 'wiki':
        return <textarea id={id} value={value} rows={field.height > 0 ? field.height : 10} disabled={disabled} onChange={e => onChange(field.name, e.target.value)} />
      case 'password':
        return <input id={id} type="password" value={value} disabled={disabled} onChange={e => onChange(field.name, e.target.value)} />
      default:
        if (field.height > 1) {
          return <textarea id={id} value={value} rows={field.height} disabled={disabled} onChange={e => onChange(field.name, e.target.value)} />
        }
        return <input id={id} type="text" value={value} disabled={disabled} onChange={e => onChange(field.name, e.target.value)} />
    }
  }

  return (
    <Styles className="form w-form-fields">
      <tbody>
        {fields
          .filter(field => field.type !== 'hidden')
          .map(field => (
            <tr key={field.name}>
              <td className="form-label">
                <label htmlFor={`form-field-${field.name}`}>{field.label || field.name}</label>
              </td>
              <td>
                {control(field)}
                {field.hint && <div className="form-hint">{field.hint}</div>}
              </td>
            </tr>
          ))}
      </tbody>
    </Styles>
  )
}

export default ArticleFormFields
