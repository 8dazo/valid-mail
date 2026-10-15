import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";
import {
  ArrowRight,
  ArrowUpRight,
  Braces,
  Check,
  ChevronDown,
  Copy,
  Download,
  FileText,
  GitFork,
  LockKeyhole,
  MailCheck,
  Sparkles,
} from "lucide-react";

type OpenTool = (id: string) => void;

function ToolkitArt({ kind }: { kind: "check" | "tidy" | "make" }) {
  const art = {
    check: "/images/editorial/tool-email.webp",
    tidy: "/images/editorial/tool-code.webp",
    make: "/images/editorial/tool-qr.webp",
  }[kind];

  return (
    <div
      className={`toolkit-art-scene toolkit-art-scene--${kind}`}
      aria-hidden="true"
    >
      <svg className="toolkit-orbit" viewBox="0 0 320 210" fill="none">
        <ellipse
          cx="158"
          cy="110"
          rx="126"
          ry="69"
          stroke="#B6A0CD"
          strokeWidth="1.5"
          strokeDasharray="4 8"
          opacity=".63"
        />
        <path d="M47 47l4 11 11 4-11 4-4 11-4-11-11-4 11-4z" fill="#AD91CF" />
        <circle cx="276" cy="75" r="7" fill="#B7DCC8" />
        <circle cx="73" cy="162" r="5" fill="#E7B8CB" />
        <path d="M270 138l6 13 13 6-13 6-6 13-6-13-13-6 13-6z" fill="#DFC6F4" />
        {kind === "check" ? (
          <path
            d="m258 32 7 7 14-15"
            stroke="#7FBA92"
            strokeWidth="3"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        ) : kind === "tidy" ? (
          <path
            d="M247 34h28M247 41h19M247 48h24"
            stroke="#9E82BF"
            strokeWidth="3"
            strokeLinecap="round"
          />
        ) : (
          <g stroke="#B593D9" strokeWidth="2">
            <path d="M259 28v20m-10-10h20M77 20v12m-6-6h12" />
          </g>
        )}
      </svg>
      <img src={art} alt="" loading="lazy" decoding="async" />
    </div>
  );
}

/**
 * A tiny illustrated network of useful tools. The shapes are intentionally SVG,
 * so they stay crisp on retina screens without loading a second hero image.
 */
export function HeroSignalArt() {
  return (
    <svg
      className="hero-signal-art"
      viewBox="0 0 420 390"
      fill="none"
      aria-hidden="true"
    >
      <defs>
        <linearGradient id="heroSignalPaper" x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#ffffff" />
          <stop offset="1" stopColor="#faf3ff" />
        </linearGradient>
        <linearGradient id="heroSignalDisc" x1="0" y1="0" x2="1" y2="1">
          <stop stopColor="#c8b1f6" />
          <stop offset="1" stopColor="#8a72c3" />
        </linearGradient>
        <filter
          id="heroSignalShadow"
          x="-50%"
          y="-50%"
          width="200%"
          height="220%"
        >
          <feDropShadow
            dx="0"
            dy="15"
            stdDeviation="16"
            floodColor="#30234d"
            floodOpacity=".19"
          />
        </filter>
      </defs>
      <circle
        cx="210"
        cy="192"
        r="151"
        stroke="#fff"
        strokeOpacity=".45"
        strokeDasharray="3 9"
      />
      <circle cx="210" cy="192" r="115" stroke="#fff" strokeOpacity=".52" />
      <circle
        cx="210"
        cy="192"
        r="78"
        fill="#ffffff"
        fillOpacity=".12"
        stroke="#ffffff"
        strokeOpacity=".5"
      />
      <path
        className="hero-signal-route"
        d="M105 131C151 68 279 65 324 139S321 286 238 307 82 254 105 131Z"
        stroke="#fff"
        strokeOpacity=".7"
        strokeWidth="2"
        strokeDasharray="7 12"
      />
      <g className="hero-signal-center" filter="url(#heroSignalShadow)">
        <rect
          x="157"
          y="139"
          width="106"
          height="106"
          rx="33"
          fill="url(#heroSignalPaper)"
        />
        <rect x="164" y="146" width="92" height="92" rx="27" stroke="#e7dafa" />
        <path
          d="M181 189h55m-55 15h36"
          stroke="#c5b1e6"
          strokeWidth="7"
          strokeLinecap="round"
        />
        <circle cx="225" cy="174" r="19" fill="#e0f3e6" />
        <path
          d="m215 174 7 7 13-15"
          stroke="#4f916d"
          strokeWidth="4.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
      <g
        className="hero-signal-card hero-signal-card-one"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="20"
          y="92"
          width="120"
          height="77"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="35" y="110" width="28" height="28" rx="9" fill="#e1d5f9" />
        <path
          d="m41 119 8 6 8-6m-16 0v10h16v-10"
          stroke="#8568bb"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M73 117h50M73 128h32"
          stroke="#c7b9de"
          strokeWidth="5"
          strokeLinecap="round"
        />
        <rect x="35" y="146" width="88" height="7" rx="3.5" fill="#e9e1f4" />
      </g>
      <g
        className="hero-signal-card hero-signal-card-two"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="292"
          y="130"
          width="104"
          height="85"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="307" y="145" width="74" height="55" rx="10" fill="#e9f4e8" />
        <path
          d="m322 171 12 10 20-23"
          stroke="#75a97d"
          strokeWidth="6"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle cx="363" cy="165" r="4" fill="#b3cdb4" />
      </g>
      <g
        className="hero-signal-card hero-signal-card-three"
        filter="url(#heroSignalShadow)"
      >
        <rect
          x="91"
          y="270"
          width="139"
          height="71"
          rx="18"
          fill="url(#heroSignalPaper)"
        />
        <rect x="105" y="283" width="39" height="42" rx="10" fill="#f4e1d9" />
        <path
          d="m127 291-8 13 8 13m-2-13h12"
          stroke="#b78875"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M153 292h61m-61 12h46m-46 12h55"
          stroke="#d7c8e9"
          strokeWidth="5"
          strokeLinecap="round"
        />
      </g>
      <g className="hero-signal-spark">
        <path d="m334 57 7 16 16 7-16 7-7 16-7-16-16-7 16-7z" fill="#f9e2a9" />
        <circle cx="65" cy="225" r="7" fill="#d2edce" />
        <circle cx="282" cy="310" r="5" fill="#ebcde1" />
        <path d="m52 51 4 7 8 4-8 4-4 8-4-8-7-4 7-4z" fill="#ffebd2" />
      </g>
    </svg>
  );
}

export function LandingIntro({
  count,
  localCount,
  onOpen,
}: {
  count: number;
  localCount: number;
  onOpen: OpenTool;
}) {
  const reduced = useReducedMotion();
  return (
    <div className="studio-landing">
      <section
        className="studio-stats container"
        aria-label="The toolkit at a glance"
      >
        {[
          [`${count}`, "Everyday tools", "One place for the little things."],
          [
            `${localCount}`,
            "Work locally",
            "Your inputs stay in your browser.",
          ],
          ["0", "Accounts needed", "Skip the setup. Start creating."],
          ["100%", "Free to use", "A good shortcut for everyone."],
        ].map(([value, title, text]) => (
          <motion.div
            key={title}
            initial={reduced ? false : { opacity: 0, y: 18 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.3 }}
            transition={{
              duration: 0.5,
              delay:
                [
                  "Everyday tools",
                  "Work locally",
                  "Accounts needed",
                  "Free to use",
                ].indexOf(title) * 0.06,
            }}
          >
            <strong>{value}</strong>
            <h3>{title}</h3>
            <p>{text}</p>
          </motion.div>
        ))}
      </section>
      <section
        className="studio-features container"
        id="how-it-works"
        aria-labelledby="features-title"
      >
        <div className="studio-section-heading">
          <div>
            <span className="studio-kicker">Make the everyday easier</span>
            <h2 id="features-title">Do the small things well.</h2>
          </div>
          <p>
            From a messy bit of data to a finished little idea.
            <br />
            Pick a tool, bring your input, make it yours.
          </p>
        </div>
        <div className="studio-feature-grid">
          <motion.article
            className="studio-feature peach"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img
                src="/images/editorial/feature-check.webp"
                alt=""
                loading="lazy"
                decoding="async"
              />
              <span className="editorial-feature-tag">
                <MailCheck size={14} /> Check an address
              </span>
            </div>
            <span className="feature-step">Check</span>
            <h3>A little more certainty.</h3>
            <p>
              Check email addresses, inspect links and understand what’s behind
              your data.
            </p>
            <button onClick={() => onOpen("email")}>
              Check an email <ArrowUpRight size={16} />
            </button>
          </motion.article>
          <motion.article
            className="studio-feature sage"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img
                src="/images/editorial/feature-tidy.webp"
                alt=""
                loading="lazy"
                decoding="async"
              />
              <span className="editorial-feature-tag">
                <Braces size={14} /> Tidy your data
              </span>
            </div>
            <span className="feature-step">Tidy</span>
            <h3>Give the mess some shape.</h3>
            <p>
              Format, convert and preview. Turn the things you’re working on
              into something useful.
            </p>
            <button onClick={() => onOpen("json")}>
              Tidy some JSON <ArrowUpRight size={16} />
            </button>
          </motion.article>
          <motion.article
            className="studio-feature rose"
            initial={reduced ? false : { opacity: 0, y: 26 }}
            whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
            viewport={{ once: true, amount: 0.15 }}
            transition={{ duration: 0.55, ease: "easeOut" }}
          >
            <div className="feature-mini editorial-feature-art">
              <img
                src="/images/editorial/feature-make.webp"
                alt=""
                loading="lazy"
                decoding="async"
              />
              <span className="editorial-feature-tag">
                <FileText size={14} /> Make something new
              </span>
            </div>
            <span className="feature-step">Make</span>
            <h3>Make it. Then make it yours.</h3>
            <p>
              Create a code, a color or a fresh idea. Copy your result or
              download it for later.
            </p>
            <button onClick={() => onOpen("qr")}>
              Make a QR code <ArrowUpRight size={16} />
            </button>
          </motion.article>
        </div>
      </section>
      <section
        className="studio-human-connection"
        aria-labelledby="human-connection-title"
      >
        <div className="human-connection-copy container">
          <span className="studio-kicker">The human side of useful tools</span>
          <h2 id="human-connection-title">
            Made for your ideas.
            <br />
            Not the other way around.
          </h2>
          <p>
            The best tools get out of your way, so you can focus on what
            matters: making something good.
          </p>
        </div>
        <div className="human-connection-panorama" aria-hidden="true">
          <img
            src="/images/editorial/hands-wide.webp"
            alt=""
            loading="lazy"
            decoding="async"
          />
        </div>
      </section>
      <section className="studio-tour container" aria-labelledby="tour-title">
        <span className="studio-kicker">See it in action</span>
        <h2 id="tour-title">From messy to readable.</h2>
        <p>Useful controls. Clear results. A shortcut that feels like yours.</p>
        <motion.div
          className="tour-frame"
          initial={reduced ? false : { opacity: 0, y: 34, scale: 0.985 }}
          whileInView={reduced ? undefined : { opacity: 1, y: 0, scale: 1 }}
          viewport={{ once: true, amount: 0.2 }}
          transition={{ duration: 0.7, ease: "easeOut" }}
        >
          <div className="tour-browser">
            <div className="tour-browser-bar">
              <span>
                <i />
                <i />
                <i />
              </span>
              <span>your little workspace</span>
              <LockKeyhole size={13} />
            </div>
            <div className="tour-app">
              <div className="tour-heading">
                <span>
                  <Braces size={19} /> JSON formatter
                </span>
                <span className="tour-local">
                  Runs locally <span />
                </span>
              </div>
              <div className="tour-panels">
                <div>
                  <span className="tour-panel-label">A little messy</span>
                  <pre>
                    {'{name:"WeMe", tools:' + count + ",\n ready:true,}"}
                  </pre>
                  <span className="tour-hint">
                    Bring what you’re working on.
                  </span>
                </div>
                <div>
                  <span className="tour-panel-label">
                    A little clearer <Check size={14} />
                  </span>
                  <pre>
                    {'{\n  "name": "WeMe",\n  "tools": ' +
                      count +
                      ',\n  "ready": true\n}'}
                  </pre>
                  <span className="tour-hint">
                    <Copy size={13} /> Copy <Download size={13} /> Download
                  </span>
                </div>
              </div>
            </div>
          </div>
          <svg
            className="tour-editorial-symbol"
            viewBox="0 0 112 112"
            aria-hidden="true"
          >
            <circle
              cx="56"
              cy="56"
              r="45"
              fill="#FAF5FF"
              stroke="#CBB7E4"
              strokeWidth="1.5"
            />
            <circle
              cx="56"
              cy="56"
              r="33"
              fill="none"
              stroke="#D4C5E7"
              strokeWidth="1.3"
              strokeDasharray="3 5"
            />
            <path
              d="M43 42c-7 0-7 7-7 13s-3 8-7 8c4 0 7 2 7 8s0 13 7 13m26-42c7 0 7 7 7 13s3 8 7 8c-4 0-7 2-7 8s0 13-7 13"
              fill="none"
              stroke="#8F6AB8"
              strokeWidth="4.2"
              strokeLinecap="round"
            />
            <path d="m57 22 3 7 7 3-7 3-3 7-3-7-7-3 7-3z" fill="#AF96CF" />
          </svg>
          <div className="tour-sticker">
            <Sparkles size={20} /> Less busywork.
            <br />
            More good stuff.
          </div>
        </motion.div>
        <button className="button studio-button" onClick={() => onOpen("json")}>
          Try the workspace <ArrowRight size={16} />
        </button>
      </section>
      <section className="studio-picks container" aria-labelledby="picks-title">
        <div className="studio-section-heading">
          <div>
            <span className="studio-kicker">Somewhere to begin</span>
            <h2 id="picks-title">Pick up where you left off.</h2>
          </div>
          <p>
            A few good places to start.
            <br />
            The rest is up to you.
          </p>
        </div>
        <div className="studio-pick-grid">
          {[
            {
              kind: "check" as const,
              id: "email",
              title: "A little confidence",
              text: "Check an address before you hit send.",
              action: "Verify an email",
            },
            {
              kind: "tidy" as const,
              id: "markdown",
              title: "A little clarity",
              text: "Give your next README a proper preview.",
              action: "Preview Markdown",
            },
            {
              kind: "make" as const,
              id: "qr",
              title: "A little possibility",
              text: "Turn a link into something worth sharing.",
              action: "Create a QR code",
            },
          ].map((item) => (
            <motion.button
              className="studio-pick"
              key={item.id}
              initial={reduced ? false : { opacity: 0, y: 24 }}
              whileInView={reduced ? undefined : { opacity: 1, y: 0 }}
              viewport={{ once: true, amount: 0.2 }}
              transition={{ duration: 0.5, ease: "easeOut" }}
              onClick={() => onOpen(item.id)}
            >
              <span className="pick-title">
                {item.title}
                <ArrowUpRight size={17} />
              </span>
              <ToolkitArt kind={item.kind} />
              <span className="pick-description">{item.text}</span>
              <span className="pick-action">
                {item.action} <ArrowRight size={14} />
              </span>
            </motion.button>
          ))}
        </div>
      </section>
    </div>
  );
}

export function LandingOutro({
  count,
  localCount,
  onSection,
}: {
  count: number;
  localCount: number;
  onSection: (id: string) => void;
}) {
  return (
    <div className="studio-landing">
      <section
        className="studio-pass"
        id="open-source"
        aria-labelledby="pass-title"
      >
        <div className="studio-pass-inner container">
          <span className="studio-kicker">One plan for everyone</span>
          <h2 id="pass-title">
            Good tools are for <em>everyone.</em>
          </h2>
          <p>
            One generous plan. No subscriptions, no sign-up.
            <br />
            Just a little help when you need it.
          </p>
          <div className="pass-card-stage">
            <div className="pass-card">
              <div className="pass-card-paper">
                <span className="pass-spark">
                  <Sparkles size={17} /> WeMe, all yours
                </span>
                <div className="pass-plan-head">
                  <span className="pass-plan-label">
                    ONE PLAN. ALL YOUR TOOLS.
                  </span>
                  <strong>Free</strong>
                  <span className="pass-plan-period">₹0 · forever</span>
                </div>
                <div className="pass-plan-divider" />
                <ul>
                  <li>
                    <span className="pass-feature-icon">
                      <Check size={14} />
                    </span>{" "}
                    All {count} tools in one collection
                  </li>
                  <li>
                    <span className="pass-feature-icon">
                      <Check size={14} />
                    </span>{" "}
                    Examples to get you started
                  </li>
                  <li>
                    <span className="pass-feature-icon">
                      <Check size={14} />
                    </span>{" "}
                    Copy and download your results
                  </li>
                  <li>
                    <span className="pass-feature-icon">
                      <Check size={14} />
                    </span>{" "}
                    Open-source foundations
                  </li>
                </ul>
                <button
                  className="button studio-button pass-primary-action"
                  onClick={() => onSection("tools")}
                >
                  Find your next shortcut <ArrowRight size={17} />
                </button>
              </div>
            </div>
          </div>
          <a
            className="studio-source-link"
            href="https://github.com/8dazo/valid-mail"
            target="_blank"
            rel="noopener noreferrer"
          >
            Built in the open. Explore the source <GitFork size={15} />
          </a>
        </div>
      </section>
      <section
        className="studio-faq container"
        id="about"
        aria-labelledby="faq-title"
      >
        <div>
          <span className="studio-kicker">Before you begin</span>
          <h2 id="faq-title">The useful details.</h2>
          <p>
            A calmer corner for builders, writers and curious people. Here’s how
            it works.
          </p>
        </div>
        <div className="faq-list">
          {[
            {
              q: "Can I use WeMe for free?",
              a: "Yes. Every tool is free to use, with no account or subscription needed.",
            },
            {
              q: "Does my data leave my browser?",
              a: `The ${localCount} local tools run in your browser. Email verification and company-email discovery send the details needed for their checks to the Go server. Your inputs aren’t saved in browser storage.`,
            },
            {
              q: "How do I get started?",
              a: "Choose a tool, paste your input or import a supported file, then run it. Every tool has an example, and you can copy or download your result.",
            },
            {
              q: "Does email verification prove a mailbox exists?",
              a: "Address and domain checks are separate from mailbox checks. SMTP needs a reachable mail-server route; a blocked or incomplete check leaves mailbox existence unknown.",
            },
            {
              q: "Can I host WeMe myself?",
              a: "Yes. The GitHub README includes local, Docker, Render and Vercel setup instructions, along with the Go libraries used by each tool.",
            },
          ].map((item, index) => (
            <details key={item.q} open={index === 0}>
              <summary>
                {item.q}
                <ChevronDown size={16} />
              </summary>
              <p>{item.a}</p>
            </details>
          ))}
        </div>
      </section>
    </div>
  );
}

export function StudioFooter({
  brand,
  onSection,
  onOpen,
}: {
  brand: ReactNode;
  onSection: (id: string) => void;
  onOpen: OpenTool;
}) {
  return (
    <footer className="studio-footer">
      <div className="container">
        <div className="studio-footer-top">
          <div className="footer-brand">
            {brand}
            <p>
              Little tools.
              <br />A little more possibility.
            </p>
            <a
              href="https://github.com/8dazo/valid-mail"
              aria-label="WeMe on GitHub"
              target="_blank"
              rel="noopener noreferrer"
            >
              <GitFork size={18} /> Built in the open <ArrowUpRight size={14} />
            </a>
          </div>
          <div>
            <h3>The toolkit</h3>
            <button onClick={() => onSection("tools")}>
              Explore all tools
            </button>
            <button onClick={() => onOpen("email")}>Email verifier</button>
            <button onClick={() => onOpen("json")}>JSON formatter</button>
            <button onClick={() => onOpen("markdown")}>README preview</button>
          </div>
          <div>
            <h3>Around here</h3>
            <button onClick={() => onSection("how-it-works")}>
              How it works
            </button>
            <button onClick={() => onSection("about")}>About WeMe</button>
            <button onClick={() => onSection("about")}>Common questions</button>
          </div>
          <div>
            <h3>Good foundations</h3>
            <a
              href="https://github.com/8dazo/valid-mail"
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub <ArrowUpRight size={12} />
            </a>
            <a
              href="https://github.com/8dazo/valid-mail/blob/main/README.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Documentation
            </a>
            <a
              href="https://github.com/8dazo/valid-mail/blob/main/docs/third-party.md"
              target="_blank"
              rel="noopener noreferrer"
            >
              Library credits
            </a>
          </div>
        </div>
      </div>
      <div className="footer-panorama">
        <img
          className="footer-panorama-image"
          src="/images/editorial/footer-panorama.webp"
          alt=""
          loading="lazy"
          decoding="async"
        />
        <div className="container footer-panorama-content">
          <div className="footer-wordmark" aria-hidden="true">
            <span>W</span>
            <span>e</span>
            <span>M</span>
            <span>e</span>
            <i>
              <Sparkles strokeWidth={1.2} />
            </i>
          </div>
          <div className="studio-footer-bottom">
            <span>A little less friction. A little more you.</span>
            <span>© {new Date().getFullYear()} WeMe</span>
          </div>
        </div>
      </div>
    </footer>
  );
}
