import React from 'react';
import PropTypes from 'prop-types';

class JobTemplateSelector extends React.Component {
    static propTypes = {
        jobTemplateList: PropTypes.array.isRequired,
        onChange: PropTypes.func.isRequired,
        value: PropTypes.string.isRequired,
        className: PropTypes.string
    };

    constructor(props) {
        super(props);

        this.state = {
            loading: false,
            lastError: null,
            templateEntries: [] //expect an object of {key: "key", value: "value"}
        };

        this.setStatePromise = this.setStatePromise.bind(this);
    }

    setStatePromise(newState){
        return new Promise((resolve, reject)=>this.setState(newState, ()=>resolve()))
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        if(prevProps.jobTemplateList!==this.props.jobTemplateList) {
            const templateEntries = this.props.jobTemplateList.map(ent=>{ return { key: ent.JobTypeName, value: ent.Id}});
            this.setState({loading: false, lastError: null, templateEntries: templateEntries});
        }
    }

    render(){
        return <div className="job-template-selector" >
            <select value={this.props.value} onChange={this.props.onChange} id="job-template-selector">
                <option key={-1} value="00000000-0000-0000-0000-000000000000">(none)</option>
                {
                    this.state.templateEntries.map((ent,idx)=><option key={idx} value={ent.value}>{ent.key}</option>)
                }
            </select>
            {this.state.lastError ? <span className="error-text">{this.state.lastError}</span> : "" }
        </div>
    }
}

export default JobTemplateSelector;