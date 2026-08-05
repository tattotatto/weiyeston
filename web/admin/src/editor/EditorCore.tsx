import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { TitleNode } from './extensions/TitleNode'
import { ParagraphNode } from './extensions/ParagraphNode'
import { ImageNode } from './extensions/ImageNode'
import { SwiperNode } from './extensions/SwiperNode'
import { VideoNode } from './extensions/VideoNode'
import { DividerNode } from './extensions/DividerNode'
import { ColumnsNode } from './extensions/ColumnsNode'
import { ButtonNode } from './extensions/ButtonNode'
import { CardNode } from './extensions/CardNode'
import { QuoteNode } from './extensions/QuoteNode'
import { SpacerNode } from './extensions/SpacerNode'
import { FollowGuideNode } from './extensions/FollowGuideNode'
import type { Editor } from '@tiptap/core'
import type { JSONContent } from '@tiptap/core'

export interface EditorCoreProps {
  initialContent?: JSONContent
  editable?: boolean
  onUpdate?: (json: JSONContent) => void
  editorRef?: React.MutableRefObject<Editor | null>
}

const DEFAULT_CONTENT: JSONContent = {
  type: 'doc',
  content: [],
}

export function getEditorJSON(editor: Editor | null): JSONContent {
  if (!editor) return DEFAULT_CONTENT
  const json = editor.getJSON()
  // Normalise: ProseMirror omits `content` when the doc is empty
  if (!json.content) {
    return { ...json, content: [] }
  }
  return json
}

export function setEditorContent(
  editor: Editor | null,
  content: JSONContent,
): void {
  if (!editor) return
  editor.commands.setContent(content)
}

export function EditorCore({
  initialContent,
  editable = true,
  onUpdate,
  editorRef,
}: EditorCoreProps) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      TitleNode,
      ParagraphNode,
      ImageNode,
      SwiperNode,
      VideoNode,
      DividerNode,
      ColumnsNode,
      ButtonNode,
      CardNode,
      QuoteNode,
      SpacerNode,
      FollowGuideNode,
    ],
    content: initialContent ?? DEFAULT_CONTENT,
    editable,
    onUpdate: ({ editor: currentEditor }) => {
      if (onUpdate) {
        onUpdate(currentEditor.getJSON())
      }
    },
  })

  // Expose editor instance via ref
  if (editorRef && editor) {
    // eslint-disable-next-line no-param-reassign
    editorRef.current = editor
  }

  return (
    <div className="editor-core-wrapper">
      <EditorContent editor={editor} className="tiptap-editor" />
    </div>
  )
}

export default EditorCore
