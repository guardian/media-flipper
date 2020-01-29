import React from 'react';
import PropTypes from 'prop-types';

class JobTemplateSelector extends React.Component {
    static propTypes = {
        onChange: PropTypes.func.isRequired,
        value: PropTypes.string.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            templateEntries: [] //expect an object of {key: "key", value: "value"}
        };

        this.setStatePromise = this.setStatePromise.bind(this);
        this.loadData = this.loadData.bind(this);
    }

    setStatePromise(newState){
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    async loadData() {
        await this.setStatePromise({loading: true, lastError: null});

        const response = await fetch("/api/jobtemplate");
        if(response.status===200){
            const serverData = await response.json();

            const templateEntries = serverData.entries.map(ent=>{ return { key: ent.JobTypeName, value: ent.Id}});
            await this.setStatePromise({loading: false, lastError: null, templateEntries: templateEntries});
            if(templateEntries.length>0) this.props.onChange({target:{value:templateEntries[0].value}});
        } else {
            const bodyText = await response.text();

            return this.setStatePromise({loading: false, lastError: bodyText})
        }
    }

    componentDidMount() {
        this.loadData();
    }

    render(){
        return <div className="job-template-selector" >
            <select value={this.props.value} onChange={this.props.onChange} id="job-template-selector">
                {
                    this.state.templateEntries.map((ent,idx)=><option key={idx} value={ent.value}>{ent.key}</option>)
                }
            </select>
            {this.state.lastError ? <span className="error-text">{this.state.lastError}</span> : "" }
        </div>
    }
}

export default JobTemplateSelector;