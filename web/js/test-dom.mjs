export class TestElement {
  constructor(tagName) { this.tagName=tagName.toUpperCase();this.children=[];this.attributes=new Map();this.dataset={};this.listeners=new Map();this.hidden=false;this.disabled=false;this.textContent='';this.className='';this.parentNode=null }
  append(...children){for(const child of children){const value=typeof child==='string'?new TestText(child):child;value.parentNode=this;this.children.push(value)}}
  appendChild(child){this.append(child);return child}
  replaceChildren(...children){this.children=[];this.append(...children)}
  remove(){if(this.parentNode)this.parentNode.children=this.parentNode.children.filter(child=>child!==this);this.parentNode=null;this.removed=true}
  setAttribute(name,value){this.attributes.set(name,String(value))}
  getAttribute(name){return this.attributes.get(name)??null}
  addEventListener(type,handler){this.listeners.set(type,handler)}
  click(){return (this.onclick||this.listeners.get('click'))?.({preventDefault(){},currentTarget:this,target:this})}
  showModal(){this.open=true}
  close(){this.open=false;this.listeners.get('close')?.()}
  matches(selector){if(selector==='[name=review_mode]:checked')return this.name==='review_mode'&&this.checked;if(selector.startsWith('.'))return this.className.split(' ').includes(selector.slice(1));if(selector.startsWith('[')){const name=selector.slice(1,-1).split('=')[0];return name.startsWith('data-')?Object.hasOwn(this.dataset,name.slice(5)):this.attributes.has(name)}return this.tagName===selector.toUpperCase()}
  querySelector(selector){return [...this.walk()].find(node=>node.matches?.(selector))||null}
  querySelectorAll(selector){return [...this.walk()].filter(node=>node.matches?.(selector))}
  *walk(){for(const child of this.children){yield child;yield* child.walk()}}
}
class TestText extends TestElement { constructor(text){super('#text');this.textContent=text} }
export class TestDocument { constructor(){this.body=new TestElement('body')}createElement(tag){return new TestElement(tag)}createTextNode(text){return new TestText(text)} }
export const findText=(root,text)=>[root,...root.walk()].find(node=>node.textContent===text)
