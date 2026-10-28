import MarkdownIt from "markdown-it";
import hljs from "highlight.js/lib/common";

const markdown = new MarkdownIt({
  html: false,
  linkify: false,
  typographer: false,
  highlight(code, language) {
    if (language && hljs.getLanguage(language)) {
      return hljs.highlight(code, { language, ignoreIllegals: true }).value;
    }
    return "";
  },
}).disable("image");

export function renderMarkdown(source) {
  return markdown.render(source);
}
