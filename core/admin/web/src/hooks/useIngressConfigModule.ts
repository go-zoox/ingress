import { useCallback, useEffect, useMemo, useState } from 'react'
import { api } from '../api/client'
import { parseModuleDoc, stringifyModuleDoc } from '../lib/ingressModuleForms'
import { isModuleDocDirty } from '../lib/configPersistDiff'

export function useIngressConfigModule(moduleId: string) {
  const [configPath, setConfigPath] = useState('')
  const [savedYAML, setSavedYAML] = useState('')
  const [doc, setDoc] = useState<Record<string, unknown>>({})
  const [savedDoc, setSavedDoc] = useState<Record<string, unknown>>({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    setErr('')
    return api
      .getConfig()
      .then((cfg) => {
        setConfigPath(cfg.path)
        setSavedYAML(cfg.content)
        return api.configModules(cfg.content).then((modules) => {
          const mod = modules.find((m) => m.id === moduleId)
          const nextDoc = parseModuleDoc(mod?.yaml ?? '')
          setDoc(nextDoc)
          setSavedDoc(nextDoc)
          setLoading(false)
        })
      })
      .catch((e: Error) => {
        setErr(e.message)
        setLoading(false)
      })
  }, [moduleId])

  useEffect(() => {
    void load()
  }, [load])

  const dirty = useMemo(() => isModuleDocDirty(doc, savedDoc), [doc, savedDoc])

  const persist = useCallback(
    async (note: string, reload: boolean): Promise<string | null> => {
      setSaving(true)
      setErr('')
      try {
        const moduleYAML = stringifyModuleDoc(doc)
        const merged = await api.mergeConfigModule(savedYAML, moduleId, moduleYAML)
        await api.validateConfig(merged.content)
        await api.putConfig(merged.content, note)
        if (reload) await api.reload()
        setSavedYAML(merged.content)
        setSavedDoc(doc)
        return null
      } catch (e: unknown) {
        const msg = e instanceof Error ? e.message : String(e)
        setErr(msg)
        return msg
      } finally {
        setSaving(false)
      }
    },
    [doc, moduleId, savedYAML],
  )

  const save = useCallback(() => persist('save', false), [persist])
  const publish = useCallback(() => persist('publish', true), [persist])

  const buildMergedContent = useCallback(async (): Promise<string> => {
    const moduleYAML = stringifyModuleDoc(doc)
    const merged = await api.mergeConfigModule(savedYAML, moduleId, moduleYAML)
    return merged.content
  }, [doc, moduleId, savedYAML])

  return {
    configPath,
    savedYAML,
    doc,
    setDoc,
    savedDoc,
    loading,
    saving,
    err,
    setErr,
    dirty,
    load,
    save,
    publish,
    buildMergedContent,
  }
}
