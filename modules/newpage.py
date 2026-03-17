from modules import ModuleError  
from renderer import RenderContext, render_template_from_string  
from urllib.parse import quote  
  
  
def has_content():  
    return False  
  
  
def allow_api():  
    return True  
  
  
def render(context: RenderContext, params):  
    """  
    NewPage 模块  
    """  
    example = params.get('example', '')  
    category = params.get('category', '')  
    submit_text = params.get('submit', '创建页面')  
      
    new_fullname = context.path_params.get('new_fullname')  
    submitted_category = context.path_params.get('category')  
      
    if new_fullname:  
        if submitted_category:  
            context.redirect_to = f"/{submitted_category}:{quote(new_fullname)}/edit/true"  
        else:  
            context.redirect_to = f"/{quote(new_fullname)}/edit/true"  
        return ''  
      
    placeholder = f"比如说, {example}" if example else "比如说, new-page-1"  
      
    return render_template_from_string("""  
    <style>  
    .new-page-form {  
      border: 1px solid #ccc;  
      background: #eee;  
      padding: 8px;  
      text-align: center;  
    }  
    .new-page-form form {  
      display: flex;  
      flex-direction: column;  
    }  
    .new-page-form p {  
      margin: 4px 0;  
    }  
    .new-page-form p div {  
      margin: 4px 0;  
    }  
    </style>  
      
    <div class="new-page-form">  
      <form class="w-ref-form" data-target-page="component:new-page" method="get" action="">  
        <p>  
          <div>页面名称:</div>  
          <input name="new_fullname" type="text" placeholder="{{ placeholder }}" required="true">  
        </p >  
        {% if category %}  
        <input style="display: none" type="hidden" name="category" value="{{ category }}">  
        {% endif %}  
        <p>  
          <input value="{{ submit_text }}" type="submit">  
        </p >  
      </form>  
    </div>  
    """,   
    placeholder=placeholder,  
    category=category,  
    submit_text=submit_text  
    )