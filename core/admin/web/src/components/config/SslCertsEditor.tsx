import { useState } from 'react'
import {
  FormField,
  FormGrid,
  FormSection,
} from '../Form'
import {
  ConfigEntityModal,
  EntityRowActions,
  EntityTableToolbar,
} from '../ConfigEntityModal'
import {
  emptySSLForm,
  sslFromDoc,
  sslFromRow,
  sslToRow,
  type SSLForm,
} from '../../lib/configEntities'
import { obj, str } from '../../lib/ingressModuleForms'

function SSLFormFields({
  form,
  onChange,
}: {
  form: SSLForm
  onChange: (next: SSLForm) => void
}) {
  const patch = (fn: (next: SSLForm) => void) => {
    const next = { ...form }
    fn(next)
    onChange(next)
  }

  return (
    <FormGrid columns={1}>
      <FormField
        label="域名"
        keyName="domain"
        value={form.domain}
        onChange={(e) => patch((n) => { n.domain = e.target.value })}
      />
      <FormField
        label="证书路径"
        keyName="certificate"
        value={form.certificate}
        onChange={(e) => patch((n) => { n.certificate = e.target.value })}
      />
      <FormField
        label="私钥路径"
        keyName="certificate_key"
        value={form.certificate_key}
        onChange={(e) => patch((n) => { n.certificate_key = e.target.value })}
      />
    </FormGrid>
  )
}

export function SslCertsEditor({
  doc,
  onChange,
}: {
  doc: Record<string, unknown>
  onChange: (doc: Record<string, unknown>) => void
}) {
  const https = { ...obj(doc.https) }
  const ssl = sslFromDoc(doc)
  const [modalOpen, setModalOpen] = useState(false)
  const [editIndex, setEditIndex] = useState<number | null>(null)
  const [draft, setDraft] = useState<SSLForm>(emptySSLForm())

  const patchSSL = (rows: Record<string, unknown>[]) => {
    onChange({ https: { ...https, ssl: rows } })
  }

  const openAdd = () => {
    setEditIndex(null)
    setDraft(emptySSLForm())
    setModalOpen(true)
  }

  const openEdit = (index: number) => {
    setEditIndex(index)
    setDraft(sslFromRow(ssl[index]))
    setModalOpen(true)
  }

  const save = () => {
    if (!draft.domain.trim()) return
    const row = sslToRow(draft)
    const next = [...ssl]
    if (editIndex == null) next.push(row)
    else next[editIndex] = row
    patchSSL(next)
    setModalOpen(false)
  }

  const remove = (index: number) => {
    if (!window.confirm(`删除证书 ${str(ssl[index]?.domain)}？`)) return
    patchSSL(ssl.filter((_, i) => i !== index))
  }

  return (
    <>
      <FormSection title={`证书列表 (${ssl.length})`}>
        <EntityTableToolbar label="https.ssl" onAdd={openAdd} />
        <table className="data config-ssl-table">
          <thead>
            <tr>
              <th>域名</th>
              <th>证书路径</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {ssl.length === 0 ? (
              <tr>
                <td colSpan={3} className="empty-hint">
                  未配置证书，点击「添加」
                </td>
              </tr>
            ) : (
              ssl.map((row, i) => {
                const cert = obj(row.cert)
                return (
                  <tr key={`${str(row.domain)}-${i}`}>
                    <td>{str(row.domain)}</td>
                    <td>
                      <code className="path-cell">{str(cert.certificate)}</code>
                    </td>
                    <td>
                      <EntityRowActions onEdit={() => openEdit(i)} onDelete={() => remove(i)} />
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </FormSection>

      <ConfigEntityModal
        open={modalOpen}
        title={editIndex == null ? '添加证书' : '编辑证书'}
        onClose={() => setModalOpen(false)}
        onSave={save}
        disableSave={!draft.domain.trim()}
      >
        <SSLFormFields form={draft} onChange={setDraft} />
      </ConfigEntityModal>
    </>
  )
}
