<script>
  import githubMark from "./assets/github.svg";
</script>

<!-- Official asset: https://brand.github.com/GitHub_Logos.zip -->
<a class="repository-link" href="https://github.com/JeHwanYoo/jmark">
  <img src={githubMark} alt="" width="20" height="20" />
  <span>JeHwanYoo/jmark</span>
</a>

<style>
  .repository-link {
    color: #65748b;
    font-size: 14px;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    text-underline-offset: 3px;
  }
  .repository-link:hover { color: #2563eb; }
  .repository-link:focus-visible {
    outline: 2px solid #2563eb;
    outline-offset: 4px;
    border-radius: 2px;
  }
</style>
