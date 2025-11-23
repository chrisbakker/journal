declare module 'quill-better-table' {
  import Quill from 'quill';
  
  interface QuillBetterTableModule {
    keyboardBindings: any;
  }
  
  const QuillBetterTable: QuillBetterTableModule & any;
  export default QuillBetterTable;
}

declare module 'quill-better-table/dist/quill-better-table.css';
